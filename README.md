# MPM Wrapper Written in Go
A wrapper that allows you to interactively install MathWorks Products using MPM (MATLAB Package Manager.) This software is not associated with or created by MathWorks. This only supports installing MATLAB toolboxes and adjacent products. It does not support the installation of support packages and you will not be given the option to download or use offline installation files.

Usage: run the program by either double-clicking on it (if your setup supports this) or by running it through the command line. Follow the prompts as given.

If you'd like to print the version number, add the argument "-version" when starting the program.

Versions compiled are from the following platforms:

- Pop!_OS 22.04 (x64)
- CentOS 7.9 (x64)
- macOS Sonoma (ARM)

If you want a compiled released for Windows or macOS x64, please let me know.

To-do:
- Prompt for admin rights when using Windows
- Separate all MATLAB products from all Polyspace products to avoid issues later on (such as when updating)
