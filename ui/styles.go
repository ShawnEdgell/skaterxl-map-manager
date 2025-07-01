package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors (Using Nord palette inspired colors for a pleasant dark theme)
	ColorPrimary      = lipgloss.Color("#88C0D0") // light blue - general highlight
	ColorAccent       = lipgloss.Color("#BF616A") // red/rose - interactive highlight
	ColorSuccess      = lipgloss.Color("#A3BE8C") // green - success messages
	ColorWarning      = lipgloss.Color("#EBCB8B") // yellow - warning messages
	ColorError        = lipgloss.Color("#BF616A") // red/rose - error messages
	ColorText         = lipgloss.Color("#ECEFF4") // light gray - general text
	// ColorBackground = lipgloss.Color("#2E3440") // Removed: For transparent background
	ColorDarkGray     = lipgloss.Color("#4C566A") // dark gray - borders, muted text
	ColorMidGray      = lipgloss.Color("#D8DEE9") // mid gray - secondary text, borders
	ColorLightGray    = lipgloss.Color("#ABB2BF") // For some UI elements, lighter than dark gray

	// --- General / Global Styles ---
	// Overall application style with padding. Background removed for transparency.
	AppStyle = lipgloss.NewStyle().
		Padding(1, 2)
		// Background(ColorBackground) // Removed: For transparent background

	// Style for any border around the main content (e.g., around the whole app or a large panel)
	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(ColorDarkGray)

	// --- Title / Header Styles ---
	TitleStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		// Background(ColorDarkGray). // Removed: For transparent background
		Padding(0, 1).
		Bold(true).
		Align(lipgloss.Center)

	// Help Text Styles (for hints like "Press q to quit")
	HelpStyle = lipgloss.NewStyle().
		Foreground(ColorLightGray).
		Padding(0, 1).
		Italic(true)

	// Status Message Styles (general info/success updates at the bottom)
	StatusMessageStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Padding(0, 1)

	// Error Message Styles (prominent errors)
	ErrorMessageStyle = lipgloss.NewStyle().
		Foreground(ColorError).
		Padding(0, 1)

	// --- Input Field Styles ---
	// Style for prompts (e.g., "Enter your directory:")
	PromptStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary)

	// Style for the text input box itself (unfocused state)
	TextInputStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(ColorDarkGray).
		Padding(0, 1).
		Foreground(ColorText)

	// Style for the text input box when it's active/focused
	FocusedTextInputStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), true).
		BorderForeground(ColorAccent).
		Padding(0, 1).
		Foreground(ColorText)

	// --- List Item Styles ---
	// Style for the list title (e.g., "Skater XL Maps")
	ListTitleStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		// Background(ColorDarkGray). // Removed: For transparent background
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)

	// Default style for non-selected list items
	ListItemStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(ColorText)

	// Style for the currently selected list item
	SelectedItemStyle = lipgloss.NewStyle().
		PaddingLeft(0). // No extra padding here as we add ">" cursor
		Foreground(ColorAccent).
		// Background(ColorDarkGray). // Removed: For transparent background, highlight will be just foreground
		Bold(true)

	// --- ASCII Art ---
	AsciiArt = lipgloss.NewStyle().Foreground(ColorPrimary).Render(`
 ___ ____ _  _ ____ ____ ___ _    ____ _  _ ___
  |  |__| |\ | |___ |__/  |  |    |___ |\ |  |
  |  |  | | \| |___ |  \  |  |___ |___ | \|  |
`) + lipgloss.NewStyle().Foreground(ColorAccent).Render(`
         Skater XL Map Installer
`) + lipgloss.NewStyle().Foreground(ColorMidGray).Render(`
           by Shawn Edgell
`)
)