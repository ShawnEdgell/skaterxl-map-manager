package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"

	"github.com/ShawnEdgell/skaterxl-map-manager/config"
	"github.com/ShawnEdgell/skaterxl-map-manager/ui"
)

var appLogger *log.Logger
var debug = flag.Bool("debug", false, "Enable debug logging to debug.log")

func main() {
	flag.Parse()

	fmt.Println("Launching Skater XL Map Manager...")

	if *debug {
		logFilePath := "debug.log"
		logFile, err := tea.LogToFile(logFilePath, "debug")
		if err != nil {
			fmt.Printf("fatal: could not setup logging: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()
		appLogger = log.New(logFile, "[APP] ", log.Ldate|log.Ltime|log.Lshortfile)
		appLogger.Println("Bubble Tea logging enabled!")
	} else {
		appLogger = log.New(ioutil.Discard, "", 0) // Discard logs if not in debug mode
	}
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
