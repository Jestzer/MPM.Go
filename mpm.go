package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

func main() {
	var operatingSystem string
	var defaultTMP string
	var mpmDownloadPath string
	var mpmURL string
	var mpmDownloadNeeded bool
	var mpmExtractNeeded bool
	var release string
	var defaultPath string

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

	// Keeping these for now for sanity's sake.
	fmt.Println("Operating System:", operatingSystem)
	fmt.Println("MPM URL:", mpmURL)
	mpmDownloadNeeded = true
	mpmExtractNeeded = true
	red := color.New(color.FgRed).SprintFunc()

	// Figure out where you want actual MPM to go.
	for {
		fmt.Print("Enter the path to the directory where you would like MPM to download to. " +
			"Press Enter to use \"" + defaultTMP + "\"\n> ")
		fmt.Scanln(&mpmDownloadPath)

		if mpmDownloadPath == "" {
			mpmDownloadPath = defaultTMP
		} else {
			_, err := os.Stat(mpmDownloadPath)
			if os.IsNotExist(err) {
				fmt.Printf("The directory \"%s\" does not exist. Do you want to create it? (y/n)\n> ", mpmDownloadPath)
				var createDir string
				fmt.Scanln(&createDir)

				// Don't ask me why I've only put this here so far.
				// I'll probably put it in other places that don't ask for file names/paths.
				if createDir == "exit" || createDir == "Exit" || createDir == "quit" || createDir == "Quit" {
					os.Exit(0)
				}

				if createDir == "y" || createDir == "Y" {
					err := os.MkdirAll(mpmDownloadPath, 0755)
					if err != nil {
						fmt.Println(red("Failed to create the directory:", err, "Please select a different directory."))
						continue
					}
					fmt.Println("Directory created successfully.")
				} else {
					fmt.Println("Directory creation skipped. Please select a different directory.")
					continue
				}
			} else if err != nil {
				fmt.Println(red("Error checking the directory:", err, "Please select a different directory."))
				continue
			}
		}

		// Check if MPM already exists in the selected directory.
		fileName := mpmDownloadPath + "/mpm"
		_, err := os.Stat(fileName)
		for {
			if err == nil {
				fmt.Print("MPM already exists in this directory. Would you like to overwrite it? ")
				fmt.Print(red("This will also overwrite the directory 'mpm-contents' if it already exists. (y/n)\n> "))
				var overwriteMPM string
				fmt.Scanln(&overwriteMPM)
				if overwriteMPM == "n" || overwriteMPM == "N" {
					fmt.Println("Skipping download.")
					mpmDownloadNeeded = false
					if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
						unzipPath := filepath.Join(mpmDownloadPath, "mpm-contents")

						// Skip download if you want to use your existing MPM, but it is not extracted.
						if _, err := os.Stat(unzipPath); os.IsNotExist(err) {
							break
						} else {
							fmt.Println("Skipping extraction.")
							mpmExtractNeeded = false
							break
						}
					} else {
						mpmExtractNeeded = false
						break
					}
				}
				if overwriteMPM == "y" || overwriteMPM == "Y" {
					break
				} else {
					fmt.Println(red("Invalid choice. Please enter either 'y' or 'n'."))
					continue
				}
			}
		}

		// Download MPM.
		if mpmDownloadNeeded {
			fmt.Println("Beginning download of MPM. Please wait.")
			err = downloadFile(mpmURL, fileName)
			if err != nil {
				fmt.Println(red("Failed to download MPM. ", err))
				continue
			}
			fmt.Println("MPM downloaded successfully.")
		}

		// Unzip the file if using Windows or macOS.
		if mpmExtractNeeded {
			if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
				fmt.Println("Beginning extraction of MPM.")
				unzipPath := filepath.Join(mpmDownloadPath, "mpm-contents")

				// Check if the "mpm-contents" directory already exists.
				if _, err := os.Stat(unzipPath); err == nil {

					// Delete the existing "mpm-contents" directory if it's there.
					err := os.RemoveAll(unzipPath)
					if err != nil {
						fmt.Println(red("Failed to delete the existing 'mpm-contents' directory:", err))
						continue
					}
				}

				err := os.MkdirAll(unzipPath, 0755)
				if err != nil {
					fmt.Println(red("Failed to create the directory:", err))
					continue
				}

				err = unzipFile(fileName, unzipPath)
				if err != nil {
					fmt.Println(red("Failed to extract MPM:", err))
					continue
				}
				fmt.Println("MPM extracted successfully.")
			}
		}
		break
	}

	// Ask the user which release they'd like to install.
	reader := bufio.NewReader(os.Stdin)
	validReleases := []string{
		"R2017b", "R2018a", "R2018b", "R2019a", "R2019b", "R2020a", "R2020b",
		"R2021a", "R2021b", "R2022a", "R2022b", "R2023a", "R2023b",
	}
	defaultRelease := "R2023b"

	for {
		fmt.Printf("Enter which release you would like to install. Press Enter to select %s: ", defaultRelease)
		fmt.Print("\n> ")
		release, _ = reader.ReadString('\n')
		release = strings.TrimSpace(release)
		if release == "" {
			release = defaultRelease
		}

		release = strings.ToLower(release)
		found := false
		for _, validRelease := range validReleases {
			if strings.ToLower(validRelease) == release {
				release = validRelease
				found = true
				break
			}
		}

		if found {
			fmt.Println("Selected release:", release)
			break
		}

		fmt.Println(red("Invalid release. Enter a release between R2017b-R2023b."))
	}

	//Product selection.
	fmt.Print("Enter the products you would like to install. Use the same syntax as MPM to specify products. " +
		"Press Enter to install all products.\n> ")
	productsInput, _ := reader.ReadString('\n')
	productsInput = strings.TrimSpace(productsInput)

	var products []string

	if productsInput == "" {
		products = []string{"MATLAB", "MATLAB_Parallel_Server"}
	} else {
		products = strings.Fields(productsInput)
	}

	fmt.Println("Products to install:", products)

	// Set the default installation path based on your OS.
	if operatingSystem == "maci64" {
		defaultPath = "/Applications/MATLAB_" + release
	}
	if operatingSystem == "win64" {
		defaultPath = "C:\\Program Files\\MATLAB\\" + release
	}
	if operatingSystem == "glnxa64" {
		defaultPath = "/usr/local/MATLAB/" + release
	}

	fmt.Print("Enter the full path where you would like to install these products. "+
		"Press Enter to install to default path: \"", defaultPath, "\"\n> ")

	installPath, _ := reader.ReadString('\n')
	installPath = strings.TrimSpace(installPath)

	if installPath == "" {
		installPath = defaultPath
	}

	// Add some code to check the following:
	// - If you have permissions to read/write there

	fmt.Println("Installation path:", installPath)

	// Optional license file selection.
	fmt.Print("If you have a license file you'd like to include in your installation, " +
		"please provide the full path to the existing license file.\n> ")

	licensePath, _ := reader.ReadString('\n')
	licensePath = strings.TrimSpace(licensePath)

	// Add some code...
	// - To check if the file actually exists
	// - That sets licenseUse = true
	// - Does not accept ""

	var mpmFullPath string
	fmt.Println(licensePath)

	if runtime.GOOS == "darwin" {
		mpmFullPath = mpmDownloadPath + "//mpm-contents//bin//maci64//mpm"
	}
	if runtime.GOOS == "windows" {
		mpmFullPath = mpmDownloadPath + "\\mpm-contents\\bin\\win64\\mpm.exe"
	}
	if runtime.GOOS == "linux" {
		mpmFullPath = mpmDownloadPath + "/mpm"
	}
	fmt.Println(mpmFullPath)

	// Yes, this is broken. Will come back to it tomorrow.
	cmd := exec.Command(mpmFullPath, "install", "--release="+release, "--destination="+installPath, "--products")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error executing mpm:", err)
	}

	// Next steps:
	// - May need to chmod mpm on Linux. Should test this soon.
	// - Ask which products they'd like to install.
	// - Painstakingly find out all products can be installed for each release on Windows and macOS.
	// - Figure out the most efficient way to do the above, including Linux.
	// - Ask for an installation path.
	// - Ask if you want to use a license file.
	// - Kick off installation.
	// - Place the license file if you asked to use one.
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

// Function to unzip MPM, since we have to on Windows and macOS.
func unzipFile(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)

		// Reconstruct the file path on Windows to ensure proper subdirectories are created.
		// Don't know why other OSes don't need this.
		if runtime.GOOS == "windows" {
			path = filepath.Join(dest, file.Name)
			path = filepath.FromSlash(path)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return err
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}
