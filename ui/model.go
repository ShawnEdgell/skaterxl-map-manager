package ui

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput" // Keep import as textInput is still used for directory
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ShawnEdgell/skaterxl-map-manager/api"
	"github.com/ShawnEdgell/skaterxl-map-manager/config"
	"github.com/ShawnEdgell/skaterxl-map-manager/installer"
)

// Declare a logger for this package, to be set from main.go
var Logger *log.Logger = log.Default()

// Define states for our TUI
type appState int

const (
	stateLoadingMaps appState = iota
	statePromptDir
	stateMapList
	stateInstalling
	stateError
	stateExiting
)

// Define custom messages for Bubble Tea
type errMsg struct{ err error }
func (e errMsg) Error() string { return e.err.Error() }

type mapsFetchedMsg []api.Map
type installProgressMsg string
type installDoneMsg struct {
	mapName string
	err     error
}

// Item implements list.Item for our maps
type Item struct {
	mapData api.Map
}

func (i Item) FilterValue() string { return i.mapData.Name }
func (i Item) Title() string       { return i.mapData.Name }
func (i Item) Description() string {
	return fmt.Sprintf("Downloads: %d | By: %s | Summary: %s",
		i.mapData.Stats.DownloadsTotal, i.mapData.SubmittedBy.Username, i.mapData.Summary)
}

// Custom list delegate to use our styles for rendering list items
type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Title())
	var renderedStr string

	width := m.Width() - SelectedItemStyle.GetPaddingLeft() - SelectedItemStyle.GetPaddingRight()

	paddedStr := lipgloss.NewStyle().Width(width).Render(str)

	if index == m.Index() {
		renderedStr = SelectedItemStyle.Render(">" + paddedStr)
	} else {
		renderedStr = ListItemStyle.Render(" " + paddedStr)
	}
	fmt.Fprint(w, renderedStr)
}


// Model represents the state of our TUI application
type Model struct {
	state           appState
	maps            []api.Map
	mapList         list.Model
	textInput       textinput.Model // Used for directory input
	currentError    error
	statusMessage   string
	skaterXLMapsDir string
	config          *config.Config
}

// NewModel initializes the Bubble Tea model
func NewModel(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "/home/shawn/.steam/steam/steamapps/compatdata/962730/pfx/drive_c/users/steamuser/Documents/SkaterXL/Maps/"
	ti.Focus()
	ti.CharLimit = 250
	ti.Width = 80
	ti.PromptStyle = PromptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	if cfg.SkaterXLMapsDir != "" {
		ti.SetValue(cfg.SkaterXLMapsDir)
		ti.CursorEnd()
	}

	m := list.New(nil, itemDelegate{}, 0, 0)
	m.Title = "Skater XL Maps"
	m.SetShowStatusBar(true)
	m.SetFilteringEnabled(false)
	m.Styles.Title = ListTitleStyle
	m.Styles.FilterPrompt = PromptStyle // Keep style, but it won't be used
	m.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ColorAccent) // Keep style, but won't be used
	m.Styles.StatusBar = lipgloss.NewStyle().Foreground(ColorDarkGray)
	m.SetShowHelp(true)


	return Model{
		state:         stateLoadingMaps,
		textInput:     ti,
		mapList:       m,
		config:        cfg,
	}
}

// Init runs once at the start of the program
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchMapsCmd(),
		textinput.Blink,
	)
}

// Update handles messages and updates the model's state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	Logger.Printf("Update: Received message type: %T, current state: %v", msg, m.state)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		Logger.Printf("Update: WindowSizeMsg received: %+v", msg)
		hPadding := AppStyle.GetHorizontalPadding()
		vPadding := AppStyle.GetVerticalPadding()
		totalNonListHeight := lipgloss.Height(TitleStyle.Render("A")) +
                             lipgloss.Height(HelpStyle.Render("A")) +
                             lipgloss.Height(StatusMessageStyle.Render("A")) +
                             vPadding*2
		m.mapList.SetSize(msg.Width - hPadding*2, msg.Height - totalNonListHeight)
		m.textInput.Width = msg.Width - hPadding*2 - 4

	case mapsFetchedMsg:
		Logger.Printf("Update: mapsFetchedMsg received. Map count: %d", len(msg))
		m.maps = msg

		sort.Slice(m.maps, func(i, j int) bool {
			return m.maps[i].DateAdded > m.maps[j].DateAdded
		})

		items := make([]list.Item, len(m.maps))
		for i, mapData := range m.maps {
			items[i] = Item{mapData: mapData}
		}
		m.mapList.SetItems(items)

		if m.config.SkaterXLMapsDir != "" {
			m.skaterXLMapsDir = m.config.SkaterXLMapsDir
			m.statusMessage = fmt.Sprintf("Using saved maps directory.")
			m.state = stateMapList
			Logger.Printf("Update: Changed state to stateMapList (saved dir). Status: %s", m.statusMessage)
		} else {
			m.state = statePromptDir
			m.statusMessage = "Please enter your Skater XL Maps directory."
			Logger.Printf("Update: Changed state to statePromptDir (no saved dir). Status: %s", m.statusMessage)
		}

	case errMsg:
		Logger.Printf("Update: errMsg received: %v", msg.err)
		m.currentError = msg.err
		m.state = stateError

	case installProgressMsg:
		Logger.Printf("Update: installProgressMsg received: %s", string(msg))
		m.statusMessage = StatusMessageStyle.Render(string(msg))

	case installDoneMsg:
		Logger.Printf("Update: installDoneMsg received: %+v", msg)
		if msg.err != nil {
			m.currentError = msg.err
			m.state = stateError
			Logger.Printf("Update: Install failed, transitioned to stateError: %v", msg.err)
		} else {
			m.statusMessage = StatusMessageStyle.Render(fmt.Sprintf("Successfully installed %s!", msg.mapName))
			m.state = stateMapList
			Logger.Printf("Update: Install successful, transitioned to stateMapList.")
		}


	case tea.KeyMsg:
		Logger.Printf("Update: KeyMsg received: %v", msg.String())
		
		// Handle global quit keys
		switch {
		case msg.Type == tea.KeyCtrlC:
			m.state = stateExiting
			return m, tea.Quit
		case msg.String() == "q": // 'q' always quits from anywhere
			m.state = stateExiting
			return m, tea.Quit
		}

		// Handle key presses based on current state
		switch m.state {
		case statePromptDir:
			switch msg.Type {
			case tea.KeyEnter:
				inputPath := strings.TrimSpace(m.textInput.Value())
				Logger.Printf("Update: Enter pressed in statePromptDir. Input: '%s'", inputPath)
				if inputPath == "" {
					m.statusMessage = ErrorMessageStyle.Render("Directory cannot be empty.")
					return m, nil
				}
				if _, err := os.Stat(inputPath); os.IsNotExist(err) {
					m.statusMessage = ErrorMessageStyle.Render(fmt.Sprintf("Directory '%s' does not exist. Please enter a valid path.", inputPath))
					Logger.Printf("Update: Invalid directory: %s", inputPath)
					return m, nil
				} else if err != nil {
					m.statusMessage = ErrorMessageStyle.Render(fmt.Sprintf("Error accessing directory '%s': %v", inputPath, err))
					Logger.Printf("Update: Directory access error: %v", err)
					return m, nil
				}

				m.skaterXLMapsDir = inputPath
				m.config.SkaterXLMapsDir = inputPath
				if err := config.SaveConfig(m.config); err != nil {
					m.statusMessage = ErrorMessageStyle.Render(fmt.Sprintf("Error saving config: %v", err))
					Logger.Printf("Update: Error saving config: %v", err)
				} else {
					m.statusMessage = StatusMessageStyle.Render("Maps directory saved! Press 'q' to quit.")
					Logger.Printf("Update: Maps directory saved: %s", m.skaterXLMapsDir)
				}
				m.state = stateMapList
				Logger.Printf("Update: Transitioned to stateMapList after directory input.")

			default:
				m.textInput, cmd = m.textInput.Update(msg)
				cmds = append(cmds, cmd)
			}

		case stateMapList:
			// --- Filter functionality removed. Only basic list navigation and install remains. ---
			switch msg.Type {
			case tea.KeyEnter:
				selectedItem, ok := m.mapList.SelectedItem().(Item)
				if !ok {
					m.statusMessage = ErrorMessageStyle.Render("No map selected. Press up/down to select a map.")
					return m, nil
				}
				Logger.Printf("Update: Selected map '%s' (ID: %d). Preparing to install.", selectedItem.mapData.Name, selectedItem.mapData.ID)
				m.statusMessage = StatusMessageStyle.Render(fmt.Sprintf("You selected: %s. Initiating install...", selectedItem.mapData.Name))
				m.state = stateInstalling
				cmds = append(cmds, m.installMapCmd(selectedItem.mapData, m.skaterXLMapsDir))

			default: // All other keys (arrows for navigation, etc.)
				m.mapList, cmd = m.mapList.Update(msg) // List handles navigation
				cmds = append(cmds, cmd)
			}

		case stateError:
			if msg.String() == "esc" {
				m.state = stateExiting
				return m, tea.Quit
			}
		}

	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	if m.state == stateLoadingMaps {
		return "Loading Skater XL Maps..."
	}

	s := strings.Builder{}

	var statusLine = ""
	if m.statusMessage != "" && m.state != stateInstalling {
		statusLine = StatusMessageStyle.Render(m.statusMessage)
	}

	switch m.state {
	case statePromptDir:
		s.WriteString(lipgloss.NewStyle().Foreground(ColorText).Render("Enter your Skater XL 'Maps' directory:"))
		s.WriteString("\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n")
		s.WriteString(HelpStyle.Render("Press Enter to confirm, Ctrl+C to quit."))
	case stateMapList:
		s.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Render(fmt.Sprintf("Found %d maps.", len(m.maps))))
		s.WriteString("\n\n")
		// Simplified help text since filter is removed
		s.WriteString(HelpStyle.Render("Use ↑/↓ to navigate, Enter to install, q to quit."))
		s.WriteString("\n")
		s.WriteString(m.mapList.View())

	case stateInstalling:
		s.WriteString(lipgloss.NewStyle().Foreground(ColorWarning).Render("Installing map... This might take a moment."))
		s.WriteString("\n\n")
		s.WriteString(m.statusMessage)
	case stateError:
		s.WriteString(ErrorMessageStyle.Render(fmt.Sprintf("An error occurred: %s", m.currentError.Error())))
		s.WriteString("\n\n")
		s.WriteString(HelpStyle.Render("Press Esc to quit."))
	case stateExiting:
		s.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Render("Exiting..."))
	}

	if statusLine != "" && m.state != stateInstalling {
		s.WriteString("\n\n")
		s.WriteString(statusLine)
	}

	return AppStyle.Render(s.String())
}

// Bubble Tea Commands
func (m Model) fetchMapsCmd() tea.Cmd {
	return func() tea.Msg {
		maps, err := api.FetchMaps()
		if err != nil {
			return errMsg{err}
		}
		return mapsFetchedMsg(maps)
	}
}

func (m Model) installMapCmd(mapToInstall api.Map, installDir string) tea.Cmd {
	return func() tea.Msg {
		Logger.Printf("Installer: Starting install for map '%s' to '%s'", mapToInstall.Name, installDir)
		err := installer.InstallMap(mapToInstall, installDir)
		if err != nil {
			Logger.Printf("Installer: Failed to install '%s': %v", mapToInstall.Name, err)
		} else {
			Logger.Printf("Installer: Successfully installed '%s'", mapToInstall.Name)
		}
		return installDoneMsg{mapName: mapToInstall.Name, err: err}
	}
}