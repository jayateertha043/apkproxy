package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	apkFilePath string
	//UserCACert  string
	keystore     string
	storepass    string
	keyalias     string
	keypass      string
	apkToolPath  string
	keytoolPath  string
	uberSignPath string
)

type NetworkSecurityConfig struct {
	XMLName    xml.Name `xml:"network-security-config"`
	CACertList []CACert `xml:"base-config>trust-anchors"`
	Debuggable bool     `xml:"debug-overrides>trust-anchors"`
}

type CACert struct {
	Certificate string `xml:",innerxml"`
}

func init() {
	//os.Setenv("NoDefaultCurrentDirectoryInExePath", "1")
	flag.StringVar(&apkFilePath, "apk", "", "APK file path")
	//flag.StringVar(&UserCACert, "User-ca", "test", "User CA certificate file path")
	flag.StringVar(&keystore, "keystore", "apkproxy.jks", "Keystore file path")
	flag.StringVar(&storepass, "storepass", "apkproxy", "Keystore password")
	flag.StringVar(&keyalias, "keyalias", "apkproxy", "Keystore key alias")
	flag.StringVar(&keypass, "keypass", "apkproxy", "Keystore key password")
	flag.Parse()
}

func main() {
	apkToolPath = os.Getenv("APKTOOL_PATH")
	keytoolPath = os.Getenv("KEYTOOL_PATH")
	uberSignPath = os.Getenv("UBER_SIGN_PATH")
	//fmt.Println(apkToolPath, keytoolPath, uberSignPath)
	if apkToolPath == "" || keytoolPath == "" || uberSignPath == "" {
		panic("APKTOOL_PATH empty")
	}
	// Check required flags
	if apkFilePath == "" /*|| UserCACert == ""*/ || keystore == "" || storepass == "" || keyalias == "" || keypass == "" {
		fmt.Println("Missing required flags")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Decompile APK
	apkDir := decompileAPK(apkFilePath)

	// Add User CA certificate to network_security_config.xml
	nsConfigPath := apkDir + "/res/xml/network_security_config.xml"
	addUserCACertToNSConfig(nsConfigPath)

	// Modify AndroidManifest.xml
	manifestPath := apkDir + "/AndroidManifest.xml"
	modifyAndroidManifest(manifestPath)

	apkName := getFileNameWithoutExt(apkFilePath)
	outDir, err := ioutil.TempDir(".", "apkproxy-"+apkName+"-")
	if err != nil {
		panic("Unable to create output directory ...")
	}

	// Rebuild APK
	outputPath := outDir + string(os.PathSeparator) + "modded-" + apkFilePath
	fmt.Println("OutputDir:", outputPath)
	rebuildApk(apkDir, outputPath, apkToolPath)

	// Generate keystore if it doesn't exist
	if _, err := os.Stat(keystore); os.IsNotExist(err) {
		generateKeystore(keystore, keyalias, storepass)
	}
	signApk(outputPath, keystore, keyalias, storepass, keypass)
	/*
		// Sign APK
		signApk(outputPath, keystore, keyalias, keypass)
		//Zip align
		outputPathz := "./mod/moddedz-" + apkFilePath*/
	//zipAlign(outputPath, outputPathz)

	fmt.Println("Done!")
}

func decompileAPK(apkFilePath string) string {
	fmt.Println("Decompiling APK...")

	_, err := os.Open(apkFilePath)
	if err != nil {
		panic("Unable to read apk file ...")
	}

	apkDir, err := ioutil.TempDir(".", "apktool-")
	if err != nil {
		panic(err)
	}
	//defer os.RemoveAll(apkDir)

	cmd := exec.Command(apkToolPath, "d", "-f", "-o", apkDir, apkFilePath)
	//fmt.Println(cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	//fmt.Println(string(stdouterr))
	if err != nil {
		panic(err)
	}
	return apkDir
}

func addUserCACertToNSConfig(nsConfigPath string) {
	fmt.Println("Adding User CA certificate to network_security_config.xml...")

	// Check if file already exists
	if _, err := os.Stat(nsConfigPath); os.IsNotExist(err) {
		// Create file if it doesn't exist
		file, err := os.Create(nsConfigPath)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		fmt.Println("File created:", nsConfigPath)
	} else {
		err := os.Remove(nsConfigPath)
		if err != nil {
			panic(err)
		}
		// Create file if it doesn't exist
		file, err := os.Create(nsConfigPath)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		fmt.Println("File created:", nsConfigPath)
	}

	/*nsConfigBytes, err := ioutil.ReadFile(nsConfigPath)
	if err != nil {
		panic(err)
	}*/

	//nsConfig := string(nsConfigBytes)
	nsConfig := `
	<network-security-config>
    <base-config>
        <trust-anchors>
            <!-- Trust preinstalled CAs --> 
            <certificates src="system" /> 
            <!-- Additionally trust user added CAs --> 
            <certificates src="user" /> 
        </trust-anchors> 
    </base-config> 
    </network-security-config>`

	if err := ioutil.WriteFile(nsConfigPath, []byte(nsConfig), 0666); err != nil {
		panic(err)
	}
}

func modifyAndroidManifest(manifestPath string) {
	fmt.Println("Modifying AndroidManifest.xml...")

	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		panic(err)
	}

	manifest := string(manifestBytes)
	if strings.Contains(manifest, "android:networkSecurityConfig") {
		manifest = strings.ReplaceAll(manifest, "android:networkSecurityConfig=\"",
			"android:networkSecurityConfig=\"@xml/network_security_config")
	} else if strings.Contains(manifest, "<application") {
		manifest = strings.ReplaceAll(manifest, "<application",
			"<application android:networkSecurityConfig=\"@xml/network_security_config\" ")
	} else {
		panic("No <application> element found in AndroidManifest.xml")
	}

	if err := ioutil.WriteFile(manifestPath, []byte(manifest), 0666); err != nil {
		panic(err)
	}
}

func rebuildApk(apkPath string, outputPath string, apktoolPath string) {
	fmt.Println("Rebuilding APK...")

	cmd := exec.Command(apktoolPath, "b", apkPath, "-o", outputPath, "--use-aapt2")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	//fmt.Println(string(stdouterr))
	if err != nil {
		panic(err)
	}

}

func signApk(apkPath string, keystorePath string, alias string, kspassword string, kskeypassword string) {
	fmt.Println("Signing APK...")
	outPutDir := filepath.Dir(apkPath) + string(os.PathSeparator) + "signed" + string(os.PathSeparator)
	fmt.Println("Signed Path:", outPutDir)
	err1 := os.Mkdir(outPutDir, 0755)
	if err1 != nil {
		panic("Unable to create signed output directory...")
	}
	// Build Uber APK Sign command
	cmd := exec.Command("java", "-jar", uberSignPath, "-a", apkPath, "--ks", keystorePath, "--ksAlias", alias, "--ksPass", kspassword, "--ksKeyPass", kskeypassword, "-out", outPutDir)
	fmt.Println(cmd.Args)
	// Run Uber APK Sign command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	//fmt.Println(string(stdouterr))
	if err != nil {
		panic(err)
	}
}

func generateKeystore(keystorePath string, alias string, password string) {
	fmt.Println("Generating keystore...")

	// Build keytool command
	cmd := exec.Command(keytoolPath, "-genkey", "-v", "-dname", "cn=apkproxy, ou=apkproxy, o=jayateertha043, c=IN", "-validity", "20000", "-keyalg", "RSA", "-keystore", keystorePath, "-alias", alias, "-storepass", password, "-keypass", password)

	// Run keytool command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	//fmt.Println(string(stdouterr))
	if err != nil {
		panic(err)
	}
}

func getFileNameWithoutExt(filePath string) string {
	fileExt := filepath.Ext(filePath)
	fileName := filepath.Base(filePath)
	fileNameWithoutExt := fileName[0 : len(fileName)-len(fileExt)]
	return fileNameWithoutExt
}
