package ui

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ShawnEdgell/skaterxl-map-manager/api"
	"github.com/ShawnEdgell/skaterxl-map-manager/config"
	"github.com/ShawnEdgell/skaterxl-map-manager/installer"
)

var Logger *log.Logger = log.Default()

type appState int

const (
	stateLoadingMaps appState = iota
	statePromptDir
	stateMapList
	stateInstalling
	stateError
	stateExiting
)

type errMsg struct{ err error }
func (e errMsg) Error() string { return e.err.Error() }

type mapsFetchedMsg []api.Map
type installProgressMsg string
type installDoneMsg struct {
	mapName string
	err     error
}

type Item struct {
	mapData api.Map
}

func (i Item) FilterValue() string { return i.mapData.Name }
func (i Item) Title() string       { return i.mapData.Name }
func (i Item) Description() string {
	return fmt.Sprintf("Downloads: %d | By: %s | Summary: %s",
		i.mapData.Stats.DownloadsTotal, i.mapData.SubmittedBy.Username, i.mapData.Summary)
}

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


type Model struct {
	state           appState
	maps            []api.Map
	mapList         list.Model
	textInput       textinput.Model
	currentError    error
	statusMessage   string
	skaterXLMapsDir string
	config          *config.Config
	sortField       string
	sortAscending   bool
}

const (
	sortByName        = "name"
	sortByPopularity  = "popularity"
	sortByRecent      = "recent"
)

func NewModel(cfg *config.Config) Model {
	ti := textinput.New()
	tti.Placeholder = "~/.steam/steam/steamapps/compatdata/962730/pfx/drive_c/users/steamuser/Documents/SkaterXL/Maps/"
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
	m.Styles.FilterPrompt = PromptStyle
	m.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ColorAccent)
	m.Styles.StatusBar = lipgloss.NewStyle().Foreground(ColorDarkGray)
	m.SetShowHelp(true)


	return Model{
		state:         stateLoadingMaps,
		textInput:     ti,
		mapList:       m,
		config:        cfg,
		sortField:     sortByRecent,
		sortAscending: false,
	}
}

func (m *Model) sortMaps() {
	sort.Slice(m.maps, func(i, j int) bool {
		switch m.sortField {
		case sortByName:
			if m.sortAscending {
				return strings.ToLower(m.maps[i].Name) < strings.ToLower(m.maps[j].Name)
			}
			return strings.ToLower(m.maps[i].Name) > strings.ToLower(m.maps[j].Name)
		case sortByPopularity:
			if m.sortAscending {
				return m.maps[i].Stats.DownloadsTotal < m.maps[j].Stats.DownloadsTotal
			}
			return m.maps[i].Stats.DownloadsTotal > m.maps[j].Stats.DownloadsTotal
		case sortByRecent:
			if m.sortAscending {
				return m.maps[i].DateAdded < m.maps[j].DateAdded
			}
			return m.maps[i].DateAdded > m.maps[j].DateAdded
		default:
			return false
		}
	})
}


func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchMapsCmd(),
		textinput.Blink,
	)
}

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
		m.mapList.SetSize(msg.Width-hPadding*2, msg.Height-totalNonListHeight)
		m.textInput.Width = msg.Width - hPadding*2 - 4

	case mapsFetchedMsg:
		Logger.Printf("Update: mapsFetchedMsg received. Map count: %d", len(msg))

		var filteredMaps []api.Map
		for _, mapData := range msg {
			lowerCaseName := strings.ToLower(mapData.Name)
			if !strings.Contains(lowerCaseName, "ps4") &&
				!strings.Contains(lowerCaseName, "playstation") &&
				!strings.Contains(lowerCaseName, "xbox") {
				filteredMaps = append(filteredMaps, mapData)
			}
		}
		m.maps = filteredMaps
		m.sortMaps()

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

		switch {
		case msg.Type == tea.KeyCtrlC:
			m.state = stateExiting
			return m, tea.Quit
		case msg.String() == "q":
			m.state = stateExiting
			return m, tea.Quit
		}

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
					return m, nil
				} else if err != nil {
					m.statusMessage = ErrorMessageStyle.Render(fmt.Sprintf("Error accessing directory '%s': %v", inputPath, err))
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
			switch key := msg.String(); key {
			case "enter":
				selectedItem, ok := m.mapList.SelectedItem().(Item)
				if !ok {
					m.statusMessage = ErrorMessageStyle.Render("No map selected. Press up/down to select a map.")
					return m, nil
				}
				Logger.Printf("Update: Selected map '%s' (ID: %d). Preparing to install.", selectedItem.mapData.Name, selectedItem.mapData.ID)
				m.statusMessage = StatusMessageStyle.Render(fmt.Sprintf("You selected: %s. Initiating install...", selectedItem.mapData.Name))
				m.state = stateInstalling
				cmds = append(cmds, m.installMapCmd(selectedItem.mapData, m.skaterXLMapsDir))

			case "1":
				switch m.sortField {
				case sortByRecent:
					m.sortField = sortByPopularity
					m.sortAscending = false
				case sortByPopularity:
					m.sortField = sortByName
					m.sortAscending = true
				case sortByName:
					m.sortField = sortByRecent
					m.sortAscending = false
				}
				m.sortMaps()
				items := make([]list.Item, len(m.maps))
				for i, mapData := range m.maps {
					items[i] = Item{mapData: mapData}
				}
				m.mapList.SetItems(items)
				m.mapList.Paginator.Page = 0
				m.mapList.Select(0)
				m.statusMessage = fmt.Sprintf("Sorted by %s (%s).", m.sortField, m.sortOrderString())

			case "2":
				m.sortAscending = !m.sortAscending
				m.sortMaps()
				items := make([]list.Item, len(m.maps))
				for i, mapData := range m.maps {
					items[i] = Item{mapData: mapData}
				}
				m.mapList.SetItems(items)
				m.mapList.Paginator.Page = 0
				m.mapList.Select(0)
				m.statusMessage = fmt.Sprintf("Sorted by %s (%s).", m.sortField, m.sortOrderString())

			default:
				m.mapList, cmd = m.mapList.Update(msg)
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
		sortOrder := m.sortOrderString()
		s.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Render(fmt.Sprintf("Found %d maps. Sorting by %s (%s).", len(m.maps), m.sortField, sortOrder)))
		s.WriteString("\n\n")
						s.WriteString(HelpStyle.Render("Use ↑/↓ to navigate, Enter to install, q to quit. Sort: (1) Cycle sort field, (2) Swap asc/desc."))
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

func (m *Model) sortOrderString() string {
	if m.sortAscending {
		return "asc"
	}
	return "desc"
}