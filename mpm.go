package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

func main() {
	var operatingSystem string
	var defaultTMP string
	var mpmURL string

	// Figure out your OS.
	switch userOS := runtime.GOOS; userOS {
	case "darwin":
		operatingSystem = "maci64"
		defaultTMP = "/tmp"
		mpmURL = "https://www.mathworks.com/mpm/maci64/mpm"
		fmt.Println("macOS")
	case "windows":
		operatingSystem = "win64"
		defaultTMP = os.Getenv("TMP")
		mpmURL = "https://www.mathworks.com/mpm/win64/mpm"
		fmt.Println("Windows")
	case "linux":
		operatingSystem = "glnxa64"
		defaultTMP = "/tmp"
		mpmURL = "https://www.mathworks.com/mpm/glxna64/mpm"
		fmt.Println("Linux")
	default:
		operatingSystem = "unknown"
	}

	if operatingSystem == "unknown" {
		fmt.Println("Your operating system is unrecognized. Exiting.")
		return
	}

	// Keeping these throughout for sanity's sake.
	fmt.Println("Operating System:", operatingSystem)
	fmt.Println("MPM URL:", mpmURL)

	// Figure out where you want actual MPM to go to.
	for {
		fmt.Print("Enter the path to the directory where you would like MPM to download to. Press Enter to use \"" + defaultTMP + "\": ")
		var mpmDownloadPath string
		fmt.Scanln(&mpmDownloadPath)

		if mpmDownloadPath == "" {
			mpmDownloadPath = defaultTMP
		} else {
			_, err := os.Stat(mpmDownloadPath)
			if os.IsNotExist(err) {
				fmt.Printf("The directory \"%s\" does not exist. Do you want to create it? (y/n): ", mpmDownloadPath)
				var createDir string
				fmt.Scanln(&createDir)
				if createDir == "y" || createDir == "Y" {
					err := os.MkdirAll(mpmDownloadPath, 0755)
					if err != nil {
						fmt.Println("Failed to create the directory:", err, "Please select a different directory.")
						continue // Go back to the beginning of the loop
					}
					fmt.Println("Directory created successfully.")
				} else {
					fmt.Println("Directory creation skipped. Please select a different directory.")
					continue // Go back to the beginning of the loop
				}
			} else if err != nil {
				fmt.Println("Error checking the directory:", err, "Please select a different directory.")
				continue // Go back to the beginning of the loop
			}
		}

		fmt.Println("Beginning download of MPM. Please wait.")

		// Check if MPM file already exists in the selected directory.
		fileName := mpmDownloadPath + "/mpm"
		_, err := os.Stat(fileName)
		if err == nil {
			fmt.Printf("MPM already exists in this directory. Would you like to overwrite it? (y/n): ")
			var overwriteMPM string
			fmt.Scanln(&overwriteMPM)
			if overwriteMPM == "n" || overwriteMPM == "N" {
				fmt.Println("Skipping download. Please select a different directory.")
				break // Hopefully your MPM isn't old junk.
			}
		}

		// Download MPM
		err = downloadFile(mpmURL, fileName)
		if err != nil {
			fmt.Println("Failed to download MPM. ", err)
			continue // Go back to the beginning of the loop
		}
		fmt.Println("MPM downloaded successfully.")

		break // Exit the loop
	}
}

// Function to download a file from the given URL and save it to the specified path.
func downloadFile(url string, filePath string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}
