<h1>apkproxy</h1>

 Rebuilds APK to allow User CA for use with MiTM tools such as Burp Suite.

## Installation

```
go install github.com/jayateertha043/apkproxy@latest
```

## Requirements
1. apktool
2. keytool
3. jarsigner


## Usage
```
Usage of apkproxy.exe:
  -apk string
        APK file path
  -keyalias string
        Keystore key alias (default "apkproxy")
  -keypass string
        Keystore key password (default "apkproxy")
  -keystore string
        Keystore file path (default "apkproxy.jks")
  -storepass string
        Keystore password (default "apkproxy")
```

## Author

👤 **Jayateertha G**

* Twitter: [@jayateerthaG](https://twitter.com/jayateerthaG)
* Github: [@jayateertha043](https://github.com/jayateertha043)
