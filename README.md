# SMM (Skater XL Map Manager)

SMM is a command-line interface (CLI) application designed to simplify the process of finding and installing maps for Skater XL. It provides a clean, interactive terminal interface to browse available maps from skatebit.app and install them directly to your game directory.

## Features

*   Browse a curated list of Skater XL maps.
*   Install maps directly to your Skater XL maps directory.
*   Simple and intuitive terminal interface.
*   Cross-platform support for Windows and Linux.

## Installation

To install SMM, ensure you have Go (version 1.16 or higher) installed on your system. You do not need to clone this repository. Simply run the following command from any directory:

```bash
go install github.com/ShawnEdgell/skaterxl-map-manager/cmd/smm
```

This command compiles the application and places the `smm` executable in your Go binary directory. Make sure this directory is in your system's `PATH` environment variable so you can run `smm` from any directory.

## Usage

Once installed, simply open your terminal and run:

```bash
smm
```

The application will guide you through setting up your Skater XL maps directory (if not already configured) and then present you with a list of available maps.

*   Use the **Up/Down arrow keys** to navigate the map list.
*   Press **Enter** to install the selected map.
*   Press **q** or **Ctrl+C** to quit the application.
*   Press **1** to cycle through sorting options (Name, Popularity, Recent).
*   Press **2** to toggle sorting order (Ascending/Descending).