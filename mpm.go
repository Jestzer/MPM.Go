package main

import (
	"fmt"
	"runtime"
)

func main() {
	os := runtime.GOOS

	switch os {
	case "darwin":
		fmt.Println("macOS")
	case "windows":
		fmt.Println("Windows")
	case "linux":
		fmt.Println("Linux")
	default:
		fmt.Println("Unknown operating system")
	}
}