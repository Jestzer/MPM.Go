package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	var operatingSystem string
	var defaultTMP string

	switch userOS := runtime.GOOS; userOS {
	case "darwin":
		operatingSystem = "maci64"
		defaultTMP = "/tmp"
		fmt.Println("macOS")
	case "windows":
		operatingSystem = "win64"
		defaultTMP = os.Getenv("TMP")
		fmt.Println("Windows")
	case "linux":
		operatingSystem = "glnxa64"
		defaultTMP = "/tmp"
		fmt.Println("Linux")
	default:
		operatingSystem = "unknown"
		defaultTMP = ""
		fmt.Println("Unknown operating system")
	}

	fmt.Println("Operating System:", operatingSystem)

	fmt.Print("Enter the path to the directory where you would like MPM to download to. Press Enter to use " + `"` + defaultTMP + `"` + ": ")
	var mpmDownloadPath string
	fmt.Scanln(&mpmDownloadPath)

	if mpmDownloadPath == "" {
		mpmDownloadPath = defaultTMP
	}

	fmt.Println("Download Path:", mpmDownloadPath)
}
