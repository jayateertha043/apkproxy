package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

var (
	apkFilePath string
	//burpCACert  string
	keystore  string
	storepass string
	keyalias  string
	keypass   string
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
	os.Setenv("NoDefaultCurrentDirectoryInExePath", "1")
	flag.StringVar(&apkFilePath, "apk", "", "APK file path")
	//flag.StringVar(&burpCACert, "burp-ca", "test", "Burp CA certificate file path")
	flag.StringVar(&keystore, "keystore", "apkproxy.jks", "Keystore file path")
	flag.StringVar(&storepass, "storepass", "apkproxy", "Keystore password")
	flag.StringVar(&keyalias, "keyalias", "apkproxy", "Keystore key alias")
	flag.StringVar(&keypass, "keypass", "apkproxy", "Keystore key password")
	flag.Parse()
}

func main() {

	// Check required flags
	if apkFilePath == "" /*|| burpCACert == ""*/ || keystore == "" || storepass == "" || keyalias == "" || keypass == "" {
		fmt.Println("Missing required flags")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Decompile APK
	apkDir := decompileAPK(apkFilePath)

	// Add Burp CA certificate to network_security_config.xml
	nsConfigPath := apkDir + "/res/xml/network_security_config.xml"
	addBurpCACertToNSConfig(nsConfigPath)

	// Modify AndroidManifest.xml
	manifestPath := apkDir + "/AndroidManifest.xml"
	modifyAndroidManifest(manifestPath)

	// Rebuild APK
	outputPath := "modded-" + apkFilePath
	rebuildApk(apkDir, outputPath, "apktool")

	// Generate keystore if it doesn't exist
	if _, err := os.Stat(keystore); os.IsNotExist(err) {
		generateKeystore(keystore, keyalias, storepass)
	}

	// Sign APK
	signApk(outputPath, keystore, keyalias, keypass)

	fmt.Println("Done!")
}

func decompileAPK(apkFilePath string) string {
	fmt.Println("Decompiling APK...")

	apkDir, err := ioutil.TempDir(".", "apktool-")
	if err != nil {
		panic(err)
	}
	//defer os.RemoveAll(apkDir)

	cmd := exec.Command("apktool.bat", "d", "-f", "-o", apkDir, apkFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	return apkDir
}

func addBurpCACertToNSConfig(nsConfigPath string) {
	fmt.Println("Adding Burp CA certificate to network_security_config.xml...")

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
			"<application android:networkSecurityConfig=\"@xml/network_security_config\"")
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
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func signApk(apkPath string, keystorePath string, alias string, password string) {
	fmt.Println("Signing APK...")

	// Build jarsigner command
	cmd := exec.Command("jarsigner", "-verbose", "-sigalg", "SHA1withRSA", "-digestalg", "SHA1",
		"-keystore", keystorePath, apkPath, alias)
	cmd.Stdin = strings.NewReader(password)

	// Run jarsigner command
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func generateKeystore(keystorePath string, alias string, password string) {
	fmt.Println("Generating keystore...")

	// Build keytool command
	cmd := exec.Command("keytool", "-genkey", "-v", "-dname", "cn=apkproxy, ou=apkproxy, o=jayateertha043, c=IN", "-validity", "20000", "-keyalg", "RSA", "-keystore", keystorePath, "-alias", alias, "-storepass", password, "-keypass", password)

	// Run keytool command
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
