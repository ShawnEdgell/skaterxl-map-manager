package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"

	"github.com/ShawnEdgell/skaterxl-map-manager/config"
	"github.com/ShawnEdgell/skaterxl-map-manager/ui"
)

var appLogger *log.Logger

func main() {
	fmt.Println("Launching Skater XL Map Manager...")

	logFilePath := "debug.log"
	logFile, err := tea.LogToFile(logFilePath, "debug")
	if err != nil {
		fmt.Printf("fatal: could not setup logging: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	appLogger = log.New(logFile, "[APP] ", log.Ldate|log.Ltime|log.Lshortfile)
	appLogger.Println("Bubble Tea logging enabled!")
	ui.Logger = appLogger

	cfg, err := config.LoadConfig()
	if err != nil {
		color.Red("Error loading configuration: %v\n", err)
		cfg = &config.Config{}
	}

	p := tea.NewProgram(ui.NewModel(cfg), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		appLogger.Fatalf("Alas, there's been an error: %v", err)
	}
}
