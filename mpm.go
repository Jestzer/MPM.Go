package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

func main() {

	var (
		defaultTMP              string
		installPath             string
		mpmDownloadPath         string
		mpmURL                  string
		mpmDownloadNeeded       bool
		products                []string
		release                 string
		defaultInstallationPath string
		licenseFileUsed         bool
		licensePath             string
		mpmFullPath             string
	)
	mpmDownloadNeeded = true
	platform := runtime.GOOS
	redText := color.New(color.FgRed).SprintFunc()
	redBackground := color.New(color.BgRed).SprintFunc()

	// Reader to make using the command line not suck.
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

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

	// Figure out your OS.
	switch platform {
	case "darwin":
		defaultTMP = "/tmp"
		switch runtime.GOARCH {
		case "amd64":
			mpmURL = "https://www.mathworks.com/mpm/maci64/mpm"
			platform = "macOSx64"
		case "arm64":
			mpmURL = "https://www.mathworks.com/mpm/maca64/mpm"
			platform = "macOSARM"
		}
	case "windows":
		defaultTMP = os.Getenv("TMP")
		mpmURL = "https://www.mathworks.com/mpm/win64/mpm"
	case "linux":
		defaultTMP = "/tmp"
		mpmURL = "https://www.mathworks.com/mpm/glnxa64/mpm"
	default:
		defaultTMP = "unknown"
		fmt.Println(redText("Your operating system is unrecognized. Exiting."))
		os.Exit(0)
	}

	// Create a valid products map for quick lookup
	validProducts := make(map[string]bool)
	for _, product := range getCompleteProductList() {
		validProducts[product] = true
	}

	// Figure out where you want actual MPM to go.
	for {
		fmt.Print("Enter the path to the directory where you would like MPM to download to. " +
			"Press Enter to use \"" + defaultTMP + "\"\n> ")
		mpmDownloadPath, err = rl.Readline()
		if err != nil {
			if err.Error() == "Interrupt" {
				fmt.Println(redText("Exiting from user input."))
			} else {
				fmt.Println(redText("Error reading line: ", err))
				continue
			}
			return
		}
		mpmDownloadPath = strings.TrimSpace(mpmDownloadPath)

		if mpmDownloadPath == "" {
			mpmDownloadPath = defaultTMP
		} else {
			_, err := os.Stat(mpmDownloadPath)
			if os.IsNotExist(err) {
				fmt.Printf("The directory \"%s\" does not exist. Do you want to create it? (y/n)\n> ", mpmDownloadPath)
				createDir, err := rl.Readline()
				if err != nil {
					if err.Error() == "Interrupt" {
						fmt.Println(redText("Exiting from user input."))
					} else {
						fmt.Println(redText("Error reading line: ", err))
						continue
					}
					return
				}
				createDir = strings.TrimSpace(createDir)

				// Don't ask me why I've only put this here so far.
				// I'll probably put it in other places that don't ask for file names/paths.
				if createDir == "exit" || createDir == "Exit" || createDir == "quit" || createDir == "Quit" {
					os.Exit(0)
				}

				if createDir == "y" || createDir == "Y" {
					err := os.MkdirAll(mpmDownloadPath, 0755)
					if err != nil {
						fmt.Println(redText("Failed to create the directory:", err, "Please select a different directory."))
						continue
					}
					fmt.Println("Directory created successfully.")
				} else {
					fmt.Println("Directory creation skipped. Please select a different directory.")
					continue
				}
			} else if err != nil {
				fmt.Println(redText("Error checking the directory:", err, "Please select a different directory."))
				continue
			}
		}

		// Check if MPM already exists in the selected directory.
		fileName := filepath.Join(mpmDownloadPath, "mpm")
		if platform == "windows" {
			fileName = filepath.Join(mpmDownloadPath, "mpm.exe")
		}
		_, err := os.Stat(fileName)
		for {
			if err == nil {
				fmt.Print("MPM already exists in this directory. Would you like to overwrite it?")
				fmt.Print(redText("This will also overwrite the directory \"mpm-contents\" and its contents if it already exists. (y/n)\n> "))
				overwriteMPM, err := rl.Readline()
				if err != nil {
					if err.Error() == "Interrupt" {
						fmt.Println(redText("Exiting from user input."))
					} else {
						fmt.Println(redText("Error reading line: ", err))
						continue
					}
					return
				}

				overwriteMPM = cleanInput(overwriteMPM)
				if overwriteMPM == "n" || overwriteMPM == "N" {
					fmt.Println("Skipping download.")
					mpmDownloadNeeded = false
				}
				if overwriteMPM == "y" || overwriteMPM == "Y" {
					break
				} else {
					fmt.Println(redText("Invalid choice. Please enter either 'y' or 'n'."))
					continue
				}
			}
			break
		}

		// Download MPM.
		if mpmDownloadNeeded {
			fmt.Println("Beginning download of MPM. Please wait.")
			err = downloadFile(mpmURL, fileName)
			if err != nil {
				fmt.Println(redText("Failed to download MPM. ", err))
				continue
			}
			fmt.Println("MPM downloaded successfully.")
		}

		// Make sure you can actually execute MPM on Linux.
		if runtime.GOOS == "linux" {
			command := "chmod +x " + mpmDownloadPath + "/mpm"

			// Execute the command
			cmd := exec.Command("bash", "-c", command)
			err := cmd.Run()

			if err != nil {
				fmt.Println("Failed to execute the command:", err)
				fmt.Print(". Either select a different directory, run this program with needed privileges, " +
					"or make modifications to MPM outside of this program.")
				continue
			}
		}
		break
	}

	// Ask the user which release they'd like to install.
	validReleases := []string{
		"R2017b", "R2018a", "R2018b", "R2019a", "R2019b", "R2020a", "R2020b",
		"R2021a", "R2021b", "R2022a", "R2022b", "R2023a", "R2023b", "R2024a",
	}
	defaultRelease := "R2024a"

	for {
		fmt.Printf("Enter which release you would like to install. Press Enter to select %s: ", defaultRelease)
		fmt.Print("\n> ")
		release, err = rl.Readline()
		if err != nil {
			if err.Error() == "Interrupt" {
				fmt.Println(redText("Exiting from user input."))
			} else {
				fmt.Println(redText("Error reading line: ", err))
				continue
			}
			return
		}

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
			break
		}

		fmt.Println(redText("Invalid release. Enter a release between R2017b-R2024a."))
	}

	for {
		// Product selection.
		fmt.Print("Enter the products you would like to install. Use the same syntax as MPM to specify products. " +
			"Press Enter to install all products.\n> ")
		productsInput, err := rl.Readline()
		if err != nil {
			if err.Error() == "Interrupt" {
				fmt.Println(redText("Exiting from user input."))
			} else {
				fmt.Println(redText("Error reading line: ", err))
				continue
			}
			return
		}

		productsInput = strings.TrimSpace(productsInput)

		if productsInput != "" && !checkForValidProducts(productsInput, validProducts) {
			fmt.Println(redText("You have entered a product that does not exist."))
			continue
		}

		// Add some code below that will break up these 2 lists between the 3 Operating Systems because right now, this only reflects Linux. Yayyyyy.
		if productsInput == "" {

			// First, filter products that have never been made available on your platform.
			if platform == "linux" {
				// products to remove: "Data_Acquisition_Toolbox", "Spreadsheet_Link",
			} else if platform == "macOSx64" {

			} else if platform == "macOSARM" {

			}

			// Next, filter productes are unavailable on your release. Some of these are just renames.

		} else if productsInput == "parallel_products" {

			products = []string{"MATLAB", "Parallel_Computing_Toolbox", "MATLAB_Parallel_Server"}

		} else {
			products = strings.Fields(productsInput)
		}
		break
	}

	// Set the default installation path based on your OS.
	if platform == "macOSx64" || platform == "macOSARM" {
		defaultInstallationPath = "/Applications/MATLAB_" + release
	}
	if platform == "windows" {
		defaultInstallationPath = "C:\\Program Files\\MATLAB\\" + release
	}
	if platform == "linux" {
		defaultInstallationPath = "/usr/local/MATLAB/" + release
	}

	for {
		fmt.Print("Enter the full path where you would like to install these products. "+
			"Press Enter to install to default path: \"", defaultInstallationPath, "\"\n> ")

		installPath, err := rl.Readline()
		if err != nil {
			if err.Error() == "Interrupt" {
				fmt.Println(redText("Exiting from user input."))
			} else {
				fmt.Println(redText("Error reading line: ", err))
				continue
			}
			return
		}
		installPath = strings.TrimSpace(installPath)

		if installPath == "" {
			installPath = defaultInstallationPath
		}
		break
	}

	// Add some code to check the following:
	// - If you have permissions to read/write there

	// Optional license file selection.
	for {
		fmt.Print("If you have a license file you'd like to include in your installation, " +
			"please provide the full path to the existing license file.\n> ")

		licensePath, err = rl.Readline()
		if err != nil {
			if err.Error() == "Interrupt" {
				fmt.Println(redText("Exiting from user input."))
			} else {
				fmt.Println(redText("Error reading line: ", err))
				continue
			}
			return
		}
		licensePath = strings.TrimSpace(licensePath)

		if licensePath == "" {
			licenseFileUsed = false
			break
		} else {

			// Check if the license file exists and has the correct extension.
			_, err := os.Stat(licensePath)
			if err != nil {
				fmt.Println(redText("Error:", err))
				continue
			} else if !strings.HasSuffix(licensePath, ".dat") && !strings.HasSuffix(licensePath, ".lic") {
				fmt.Println(redText("Invalid file extension. Please provide a file with .dat or .lic extension."))
				continue
			} else {
				licenseFileUsed = true
				break
			}
		}
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
	err = cmd.Run()
	if err != nil {
		fmt.Println(redText("Error executing MPM. See the error above for more information.", err))
	}

	// Create the licenses directory and the file specified, if you specified one.
	if licenseFileUsed {

		// Create the directory.
		licensesInstallationDirectory := filepath.Join(installPath, "licenses")
		err := os.Mkdir(licensesInstallationDirectory, 0755)
		if err != nil {
			fmt.Println(redText("Error creating \"licenses\" directory:", err))
		}

		// Copy the license file to the "licenses" directory.
		licenseFile := filepath.Base(licensePath)
		destPath := filepath.Join(licensesInstallationDirectory, licenseFile)

		src, err := os.Open(licensePath)
		if err != nil {
			fmt.Println(redText("Error opening license file:", err))
		}
		defer src.Close()

		dest, err := os.Create(destPath)
		if err != nil {
			fmt.Println(redText("Error creating destination file:", err))
		}
		defer dest.Close()

		_, err = io.Copy(dest, src)
		if err != nil {
			fmt.Println(redText("Error copying license file:", err))
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

// Clean input function.
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

// Check to make sure your entered products exist.
func checkForValidProducts(input string, validProducts map[string]bool) bool {
	products := strings.Split(input, " ")
	for _, product := range products {
		if _, exists := validProducts[product]; !exists {
			return false
		}
	}
	return true
}

// Lists

// Every product for every scenario
Aerospace_Blockset
Aerospace_Toolbox
Antenna_Toolbox
Bioinformatics_Toolbox
Control_System_Toolbox
Curve_Fitting_Toolbox
DSP_System_Toolbox
Database_Toolbox
Datafeed_Toolbox
Econometrics_Toolbox
Embedded_Coder
Filter_Design_HDL_Coder
Financial_Instruments_Toolbox
Financial_Toolbox
Fuzzy_Logic_Toolbox
GPU_Coder
Global_Optimization_Toolbox
HDL_Coder
HDL_Verifier
Image_Acquisition_Toolbox
Image_Processing_Toolbox
Instrument_Control_Toolbox
MATLAB
MATLAB_Coder
MATLAB_Compiler
MATLAB_Compiler_SDK
MATLAB_Report_Generator
Mapping_Toolbox
Model_Predictive_Control_Toolbox
Optimization_Toolbox
Parallel_Computing_Toolbox
Partial_Differential_Equation_Toolbox
Phased_Array_System_Toolbox
Powertrain_Blockset
RF_Blockset
RF_Toolbox
Risk_Management_Toolbox
Robotics_System_Toolbox
Robust_Control_Toolbox
Signal_Processing_Toolbox
SimBiology
SimEvents
Simscape
Simscape_Driveline
Simscape_Fluids
Simscape_Multibody
Simulink
Simulink_3D_Animation
Simulink_Check
Simulink_Coder
Simulink_Control_Design
Simulink_Coverage
Simulink_Design_Optimization
Simulink_Design_Verifier
Simulink_Report_Generator
Simulink_Test
Stateflow
Statistics_and_Machine_Learning_Toolbox
Symbolic_Math_Toolbox
System_Identification_Toolbox
Text_Analytics_Toolbox
Vision_HDL_Toolbox
Wavelet_Toolbox

// Every possible added product
5G_Toolbox
Audio_System_Toolbox
Audio_Toolbox
AUTOSAR_Blockset
Automated_Driving_Toolbox
Automated_Driving_System_Toolbox
Bluetooth_Toolbox
C2000_Microcontroller_Blockset
Communications_System_Toolbox
Communications_Toolbox
Computer_Vision_System_Toolbox
Computer_Vision_Toolbox
Data_Acquisition_Toolbox
DDS_Blockset
Deep_Learning_HDL_Toolbox
Deep_Learning_Toolbox
DSP_HDL_Toolbox
Fixed-Point_Designer
Fixed_Point_Designer
Industrial_Communication_Toolbox
Lidar_Toolbox
LTE_HDL_Toolbox
LTE_System_Toolbox
LTE_Toolbox
MATLAB_Distributed_Computing_Server
MATLAB_Parallel_Server
MATLAB_Production_Server
MATLAB_Test
MATLAB_Web_App_Server
Medical_Imaging_Toolbox
Mixed-Signal_Blockset
Model_Based_Calibration_Toolbox
Motor_Control_Blockset
Navigation_Toolbox
Neural_Network_Toolbox
OPC_Toolbox
Polyspace_Bug_Finder
Polyspace_Bug_Finder_Server
Polyspace_Code_Prover
Polyspace_Code_Prover_Server
Polyspace_Test
Predictive_Maintenance_Toolbox
Radar_Toolbox
Reinforcement_Learning_Toolbox
Requirements_Toolbox
RF_PCB_Toolbox
ROS_Toolbox
Satellite_Communications_Toolbox
Sensor_Fusion_and_Tracking_Toolbox
SerDes_Toolbox
Signal_Integrity_Toolbox
Simscape_Battery
Simscape_Electrical
Simscape_Electronics
Simulink_Compiler
Simulink_Desktop_Real-Time
Simulink_Desktop_Real_Time
Simulink_Fault_Analyzer
Simulink_PLC_Coder
Simscape_Power_Systems
Simulink_Real-Time
Simulink_Real_Time
Simulink_Requirements
SoC_Blockset
Spreadsheet_Link
System_Composer
Trading_Toolbox
UAV_Toolbox
Vehicle_Dynamics_Blockset
Vehicle_Network_Toolbox
Wireless_HDL_Toolbox
Wireless_Testbench
WLAN_System_Toolbox
WLAN_Toolbox

// R2017b, Universal
Audio_System_Toolbox
Automated_Driving_System_Toolbox
Communications_System_Toolbox
Computer_Vision_System_Toolbox
Fixed_Point_Designer
LTE_HDL_Toolbox
LTE_System_Toolbox
MATLAB_Distributed_Computing_Server
Model_Based_Calibration_Toolbox
Neural_Network_Toolbox
Simscape_Electronics
Simscape_Power_Systems
Simulink_Requirements
Trading_Toolbox
WLAN_System_Toolbox

// R2018a, Universal

// R2018b, Universal

// R2019a, Universal

// R2019b, Universal

// R2020a, Universal

// R2020b, Universal

// R2021a, Universal

// R2021b, Universal

// R2022a, Universal

// R2022b, Universal

// R2023a, Universal

// R2023b, Universal

// R2024a, Universal

// R2017b, Windows
OPC_Toolbox
Simulink_Desktop_Real_Time
Simulink_PLC_Coder
Simulink_Real_Time
Vehicle_Network_Toolbox

// R2018a, Windows
Vehicle_Network_Toolbox

// R2018b, Windows

// R2019a, Windows

// R2019b, Windows

// R2020a, Windows

// R2020b, Windows

// R2021a, Windows

// R2021b, Windows

// R2022a, Windows

// R2022b, Windows

// R2023a, Windows

// R2023b, Windows

// R2024a, Windows

// R2017b, Linux
Vehicle_Network_Toolbox

// R2018a, Linux
Vehicle_Network_Toolbox

// R2018b, Linux

// R2019a, Linux

// R2019b, Linux

// R2020a, Linux

// R2020b, Linux

// R2021a, Linux

// R2021b, Linux

// R2022a, Linux

// R2022b, Linux

// R2023a, Linux

// R2023b, Linux

// R2024a, Linux

// R2017b, macOSx64
Simulink_Desktop_Real_Time

// R2018a, macOSx64

// R2018b, macOSx64

// R2019a, macOSx64

// R2019b, macOSx64

// R2020a, macOSx64

// R2020b, macOSx64

// R2021a, macOSx64

// R2021b, macOSx64

// R2022a, macOSx64

// R2022b, macOSx64

// R2023a, macOSx64

// R2023b, macOSx64

// R2024a, macOSx64

// R2023b, macOSARM

// R2024a, macOSARM

// All releases, Windows
Data_Acquisition_Toolbox
Spreadsheet_Link

// All releases, Windows, Linux and macOSx64
MATLAB_Production_Server
Polyspace_Bug_Finder
Polyspace_Code_Prover