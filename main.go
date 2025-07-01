package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"

	"github.com/ShawnEdgell/skaterxl-map-manager/api"     // ADD THIS LINE
	"github.com/ShawnEdgell/skaterxl-map-manager/config"
	"github.com/ShawnEdgell/skaterxl-map-manager/installer" // ADD THIS LINE
	"github.com/ShawnEdgell/skaterxl-map-manager/ui"
)

// Declare a global logger instance
var appLogger *log.Logger

func main() {
	fmt.Println("Launching Skater XL Map Manager...")

	// --- LOGGING SETUP ---
	logFilePath := "debug.log"
	// Create or open the log file
	logFile, err := tea.LogToFile(logFilePath, "debug") // tea.LogToFile sets up default log output
	if err != nil {
		fmt.Printf("fatal: could not setup logging: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close() // Ensure the log file is closed when the program exits

	// Create a new logger that exclusively writes to our debug file
	appLogger = log.New(logFile, "[APP] ", log.Ldate|log.Ltime|log.Lshortfile)
	appLogger.Println("Bubble Tea logging enabled!")
	api.Logger = appLogger // Set the logger for the api package
	ui.Logger = appLogger  // Set the logger for the ui package
	installer.Logger = appLogger // Set the logger for the installer package (will need to add var to installer too)

	// --- END LOGGING SETUP ---

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		color.Red("Error loading configuration: %v\n", err)
		cfg = &config.Config{}
	}

	// Optional: Override default directory if provided as CLI argument
	if len(os.Args) > 1 {
		argPath := os.Args[1]
		if _, err := os.Stat(argPath); os.IsNotExist(err) {
			color.Yellow("Warning: Provided directory '%s' does not exist. Ignoring CLI argument.\n", argPath)
		} else if err != nil {
			color.Red("Warning: Error accessing provided directory '%s': %v. Ignoring CLI argument.\n", argPath, err)
		} else {
			cfg.SkaterXLMapsDir = strings.TrimSpace(argPath)
			color.Green("Using Skater XL Maps directory from CLI argument: %s\n", cfg.SkaterXLMapsDir)
		}
	}

	p := tea.NewProgram(ui.NewModel(cfg), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		appLogger.Fatalf("Alas, there's been an error: %v", err) // Use appLogger here
	}
}