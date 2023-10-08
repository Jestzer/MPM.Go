package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

func main() {

	// Setup variables that will be used across this program.
	var debug bool
	var defaultTMP string
	var mpmDownloadPath string
	var mpmURL string
	var mpmDownloadNeeded bool
	var mpmExtractNeeded bool
	var release string
	var defaultInstallationPath string
	var licenseFileUsed bool
	var licensePath string
	var mpmFullPath string
	mpmDownloadNeeded = true
	mpmExtractNeeded = true
	red := color.New(color.FgRed).SprintFunc()
	redBackground := color.New(color.BgRed).SprintFunc()
	blue := color.New(color.BgBlue).SprintFunc()
	reader := bufio.NewReader(os.Stdin)

	// Setup for better Ctrl+C messaging. This is a channel to receive OS signals.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to listen for signals.
	go func() {

		// Wait for the signal.
		<-signalChan

		// Handle the signal (in this case, simply exit the program.)
		fmt.Println(redBackground("\nExiting from user input..."))
		os.Exit(0)
	}()

	// Detect if you enabled debug mode.
	args := os.Args[1:]
	debug = false

	// The for loop that looks over any inputted arguments.
	for _, arg := range args {
		if arg == "-debug" {
			debug = true
			fmt.Println(blue("Debug mode enabled."))
			break
		}
	}

	// Figure out your OS.
	switch userOS := runtime.GOOS; userOS {
	case "darwin":
		defaultTMP = "/tmp"
		mpmURL = "https://www.mathworks.com/mpm/maci64/mpm"
		if debug {
			fmt.Println(blue("macOS"))
		}
		fmt.Println(redBackground("MPM currently requires gatekeeper to be disabled on macOS. " +
			"Please disable it before running this program, if you haven't already."))
	case "windows":
		defaultTMP = os.Getenv("TMP")
		mpmURL = "https://www.mathworks.com/mpm/win64/mpm"
		if debug {
			fmt.Println(blue("Windows"))
		}
	case "linux":
		defaultTMP = "/tmp"
		mpmURL = "https://www.mathworks.com/mpm/glnxa64/mpm"
		if debug {
			fmt.Println(blue("Linux"))
		}
	default:
		defaultTMP = "unknown"
		fmt.Println(red("Your operating system is unrecognized. Exiting."))
		os.Exit(0)
	}

	if debug {
		fmt.Println(blue("MPM URL:", mpmURL))
	}

	// Figure out where you want actual MPM to go.
	for {
		fmt.Print("Enter the path to the directory where you would like MPM to download to. " +
			"Press Enter to use \"" + defaultTMP + "\"\n> ")
		mpmDownloadPath, _ = reader.ReadString('\n')
		mpmDownloadPath = strings.TrimSpace(mpmDownloadPath)

		// Debug point 1
		if debug {
			fmt.Println(blue("1"))
		}
		if mpmDownloadPath == "" {
			mpmDownloadPath = defaultTMP
			if debug {
				fmt.Println(blue("defaultTMP: " + defaultTMP))
				fmt.Println(blue("mpmDownloadPath Line 116: " + mpmDownloadPath))
			}
		} else {
			_, err := os.Stat(mpmDownloadPath)
			if os.IsNotExist(err) {
				fmt.Printf("The directory \"%s\" does not exist. Do you want to create it? (y/n)\n> ", mpmDownloadPath)
				createDir, _ := reader.ReadString('\n')
				createDir = strings.TrimSpace(createDir)

				// Debug point 2
				if debug {
					fmt.Println(blue("2"))
				}

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

			// Debug point 3
			if debug {
				fmt.Println(blue("3"))
			}
		}
		// Debug point 4
		if debug {
			fmt.Println(blue("4"))
		}

		// Check if MPM already exists in the selected directory.
		fileName := filepath.Join(mpmDownloadPath, "mpm")
		_, err := os.Stat(fileName)
		for {
			if err == nil {
				fmt.Print("MPM already exists in this directory. Would you like to overwrite it? ")
				fmt.Print(red("This will also overwrite the directory \"mpm-contents\" and its contents if it already exists. (y/n)\n> "))
				overwriteMPM, _ := reader.ReadString('\n')
				overwriteMPM = cleanInput(overwriteMPM)
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
			//Debug point 5
			if debug {
				fmt.Println(blue("5"))
			}
			break
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
						fmt.Println(red("Failed to delete the existing \"mpm-contents\" directory:", err))
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
			if debug {
				fmt.Println(blue("mpmDownloadPath Line 246: " + mpmDownloadPath))
			}
		}
		if debug {
			fmt.Println(blue("mpmDownloadPath Line 250: " + mpmDownloadPath))
		}

		// Make sure you can actually execute MPM on Linux.
		if runtime.GOOS == "linux" {
			command := "chmod +x " + mpmDownloadPath + "/mpm"
			if debug {
				fmt.Println(blue("Command to execute: " + command))
			}

			// Execute the command
			cmd := exec.Command("bash", "-c", command)
			err := cmd.Run()

			if err != nil {
				fmt.Println("Failed to execute the command:", err)
				fmt.Print(". Either select a different directory, run this program with needed privileges, " +
					"or make modifications to MPM outside of this program.")
				continue
			}

			if debug {
				fmt.Println(blue("chmod command executed successfully."))
			}
		}
		break
	}

	if debug {
		fmt.Println(blue("mpmDownloadPath Line 239: " + mpmDownloadPath))
	}

	// Ask the user which release they'd like to install.
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

		if runtime.GOOS == "windows" && release == "R2023b" {
			fmt.Println(red("MPM currently does not support R2023b on Windows. " +
				"Please select a different release."))
			continue
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
			if debug {
				fmt.Println(blue("Selected release:", release))
			}
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

	// Add some code below that will break up these 2 lists between the 3 Operating Systems because right now, this only reflect Linux. Yayyyyy.
	if productsInput == "" {

		// This list will start from the bottom and add products as it goes up the list, stopping when it matches your release.
		// This list reflects products that have been added or renamed over time, except for R2017b, which is just the base products for MPM.
		newProductsToAdd := map[string]string{
			"R2023b": "Simulink_Fault_Analyzer Polyspace_Test Simulink_Desktop_Real-Time",
			"R2023a": "MATLAB_Test C2000_Microcontroller_Blockset",
			"R2022b": "Medical_Imaging_Toolbox Simscape_Battery",
			"R2022a": "Wireless_Testbench Simulink_Real-Time Bluetooth_Toolbox DSP_HDL_Toolbox Requirements_Toolbox Industrial_Communication_Toolbox",
			"R2021b": "Signal_Integrity_Toolbox RF_PCB_Toolbox",
			"R2021a": "Satellite_Communications_Toolbox DDS_Blockset",
			"R2020b": "UAV_Toolbox Radar_Toolbox Lidar_Toolbox Deep_Learning_HDL_Toolbox",
			"R2020a": "Simulink_Compiler Motor_Control_Blockset MATLAB_Web_App_Server Wireless_HDL_Toolbox",
			"R2019b": "ROS_Toolbox Simulink_PLC_Coder Navigation_Toolbox",
			"R2019a": "System_Composer SoC_Blockset SerDes_Toolbox Reinforcement_Learning_Toolbox Audio_Toolbox Mixed-Signal_Blockset Mixed-Signal_Blockset AUTOSAR_Blockset MATLAB_Parallel_Server Polyspace_Bug_Finder_Server Polyspace_Code_Prover_Server Automated_Driving_Toolbox Computer_Vision_Toolbox",
			"R2018b": "Communications_Toolbox Simscape_Electrical Sensor_Fusion_and_Tracking_Toolbox Deep_Learning_Toolbox 5G_Toolbox WLAN_Toolbox LTE_Toolbox",
			"R2018a": "Predictive_Maintenance_Toolbox Vehicle_Network_Toolbox Vehicle_Dynamics_Blockset",
			"R2017b": "Aerospace_Blockset Aerospace_Toolbox Antenna_Toolbox Bioinformatics_Toolbox Control_System_Toolbox Curve_Fitting_Toolbox DSP_System_Toolbox Database_Toolbox Datafeed_Toolbox Econometrics_Toolbox Embedded_Coder Filter_Design_HDL_Coder Financial_Instruments_Toolbox Financial_Toolbox Fixed-Point_Designer Fuzzy_Logic_Toolbox GPU_Coder Global_Optimization_Toolbox HDL_Coder HDL_Verifier Image_Acquisition_Toolbox Image_Processing_Toolbox Instrument_Control_Toolbox MATLAB MATLAB_Coder MATLAB_Compiler MATLAB_Compiler_SDK MATLAB_Production_Server MATLAB_Report_Generator Mapping_Toolbox Model_Predictive_Control_Toolbox Optimization_Toolbox Parallel_Computing_Toolbox Partial_Differential_Equation_Toolbox Phased_Array_System_Toolbox Polyspace_Bug_Finder Polyspace_Code_Prover Powertrain_Blockset RF_Blockset RF_Toolbox Risk_Management_Toolbox Robotics_System_Toolbox Robust_Control_Toolbox Signal_Processing_Toolbox SimBiology SimEvents Simscape Simscape_Driveline Simscape_Fluids Simscape_Multibody Simulink Simulink_3D_Animation Simulink_Check Simulink_Coder Simulink_Control_Design Simulink_Coverage Simulink_Design_Optimization Simulink_Design_Verifier Simulink_Report_Generator Simulink_Test Stateflow Statistics_and_Machine_Learning_Toolbox Symbolic_Math_Toolbox System_Identification_Toolbox Text_Analytics_Toolbox Vision_HDL_Toolbox Wavelet_Toolbox",
		}

		// The actual for loop that goes through the list above.
		for releaseLoop, product := range newProductsToAdd {
			if release >= releaseLoop {
				products = append(products, strings.Fields(product)...)
			}
		}

		// This list will start from the top and add products as it goes down the list, stopping when it matches your release.
		// This list reflects products that have been removed or renamed over time.
		oldProductsToAdd := map[string]string{
			"R2021b": "Simulink_Requirements",
			"R2020b": "Fixed-Point_Designer Trading_Toolbox",
			"R2019b": "LTE_HDL_Toolbox",
			"R2018b": "Audio_System_Toolbox Automated_Driving_System_Toolbox Computer_Vision_System_Toolbox MATLAB_Distributed_Computing_Server",
			"R2018a": "Communications_System_Toolbox LTE_System_Toolbox Neural_Network_Toolbox Simscape_Electronics Simscape_Power_Systems WLAN_System_Toolbox",
		}

		// The actual for loop that goes through the list above. Note that it uses the same logic, just <= instead of >=.
		for releaseLoop, product := range oldProductsToAdd {
			if release <= releaseLoop {
				products = append(products, strings.Fields(product)...)
			}
		}
	} else {
		products = strings.Fields(productsInput)
	}

	if debug {
		fmt.Println(blue("Products to install:", products))
	}

	// Set the default installation path based on your OS.
	if runtime.GOOS == "darwin" {
		defaultInstallationPath = "/Applications/MATLAB_" + release
	}
	if runtime.GOOS == "windows" {
		defaultInstallationPath = "C:\\Program Files\\MATLAB\\" + release
	}
	if runtime.GOOS == "linux" {
		defaultInstallationPath = "/usr/local/MATLAB/" + release
	}

	fmt.Print("Enter the full path where you would like to install these products. "+
		"Press Enter to install to default path: \"", defaultInstallationPath, "\"\n> ")

	installPath, _ := reader.ReadString('\n')
	installPath = strings.TrimSpace(installPath)

	if installPath == "" {
		installPath = defaultInstallationPath
	}

	// Add some code to check the following:
	// - If you have permissions to read/write there

	if debug {
		fmt.Println(blue("Installation path:", installPath))
	}

	// Optional license file selection.
	for {
		fmt.Print("If you have a license file you'd like to include in your installation, " +
			"please provide the full path to the existing license file.\n> ")

		licensePath, _ := reader.ReadString('\n')
		licensePath = strings.TrimSpace(licensePath)

		if licensePath == "" {
			licenseFileUsed = false
			break
		} else {

			// Check if the license file exists and has the correct extension.
			_, err := os.Stat(licensePath)
			if err != nil {
				fmt.Println(red("Error:", err))
				continue
			} else if !strings.HasSuffix(licensePath, ".dat") && !strings.HasSuffix(licensePath, ".lic") {
				fmt.Println(red("Invalid file extension. Please provide a file with .dat or .lic extension."))
				continue
			} else {
				licenseFileUsed = true
				break
			}
		}
	}

	if debug {
		fmt.Println(blue(licensePath))
	}

	if runtime.GOOS == "darwin" {
		mpmFullPath = mpmDownloadPath + "//mpm-contents//bin//maci64//mpm"
	}
	if runtime.GOOS == "windows" {
		mpmFullPath = mpmDownloadPath + "\\mpm-contents\\bin\\win64\\mpm.exe"
	}
	if runtime.GOOS == "linux" {
		mpmFullPath = mpmDownloadPath + "/mpm"
	}

	if debug {
		fmt.Println(blue(mpmFullPath))
	}

	// Construct the command and arguments to launch MPM.
	cmdArgs := []string{
		mpmFullPath,
		"install",
		"--release=" + release,
		"--destination=" + installPath,
		"--products",
	}
	cmdArgs = append(cmdArgs, products...)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(red("Error executing MPM. See the error above for more information.", err))
	}

	// Create the licenses directory and the file specified, if you specified one.
	if licenseFileUsed {

		// Create the directory.
		licensesInstallationDirectory := filepath.Join(installPath, "licenses")
		err := os.Mkdir(licensesInstallationDirectory, 0755)
		if err != nil {
			fmt.Println(red("Error creating \"licenses\" directory:", err))
		}

		// Copy the license file to the "licenses" directory.
		licenseFile := filepath.Base(licensePath)
		destPath := filepath.Join(licensesInstallationDirectory, licenseFile)

		src, err := os.Open(licensePath)
		if err != nil {
			fmt.Println(red("Error opening license file:", err))
		}
		defer src.Close()

		dest, err := os.Create(destPath)
		if err != nil {
			fmt.Println(red("Error creating destination file:", err))
		}
		defer dest.Close()

		_, err = io.Copy(dest, src)
		if err != nil {
			fmt.Println(red("Error copying license file:", err))
		}
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

// Clean input function
func cleanInput(input string) string {
	return strings.TrimSpace(input)
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
