package main

import (
	"fmt"
	"runtime"
)

func main() {
	var operatingSystem string

	switch os := runtime.GOOS; os {
	case "darwin":
		operatingSystem = "maci64"
		fmt.Println("macOS")
	case "windows":
		operatingSystem = "win64"
		fmt.Println("Windows")
	case "linux":
		operatingSystem = "glnxa64"
		fmt.Println("Linux")
	default:
		operatingSystem = "unknown"
		fmt.Println("Unknown operating system")
	}

	fmt.Println("Operating System:", operatingSystem)
}