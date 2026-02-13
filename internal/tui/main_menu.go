// Package tui provides Terminal User Interface components for gimage.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents an item in the main menu
type MenuItem struct {
	Title       string
	Description string
	Screen      Screen
}

// MainMenuModel is the model for the main menu screen
type MainMenuModel struct {
	items        []MenuItem
	selected     int
	width        int
	height       int
	showingHelp  bool
}

// NewMainMenuModel creates a new main menu model
func NewMainMenuModel() *MainMenuModel {
	return &MainMenuModel{
		items: []MenuItem{
			{
				Title:       "Generate Image",
				Description: "Create AI-generated images from text prompts",
				Screen:      ScreenGenerate,
			},
			{
				Title:       "Process Image",
				Description: "Resize, crop, compress, or convert images",
				Screen:      ScreenProcess,
			},
			{
				Title:       "Settings",
				Description: "Configure API keys and preferences",
				Screen:      ScreenSettings,
			},
			{
				Title:       "Help",
				Description: "View keyboard shortcuts and documentation",
				Screen:      ScreenHelp,
			},
		},
		selected: 0,
	}
}

// Update handles messages for the main menu
func (m *MainMenuModel) Update(msg tea.Msg) (*MainMenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case "enter", " ":
			// Navigate to selected screen
			return m, Navigate(m.items[m.selected].Screen)
		case "?":
			m.showingHelp = !m.showingHelp
		case "esc":
			if m.showingHelp {
				m.showingHelp = false
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View renders the main menu
func (m *MainMenuModel) View() string {
	if m.showingHelp {
		return m.renderHelp()
	}

	// Build the menu
	var menuItems []string

	for i, item := range m.items {
		var style lipgloss.Style
		if i == m.selected {
			style = SelectedMenuItemStyle
		} else {
			style = MenuItemStyle
		}

		cursor := "  "
		if i == m.selected {
			cursor = "> "
		}

		title := style.Render(cursor + item.Title)
		desc := MutedStyle.Render("  " + item.Description)

		menuItems = append(menuItems, title+"\n"+desc)
	}

	// Assemble the view
	logo := TitleStyle.Render(SmallerLogoASCII)
	subtitle := SubtitleStyle.Render("AI Image Generation & Processing")
	menu := strings.Join(menuItems, "\n\n")
	help := HelpStyle.Render("\n↑/↓: Navigate • Enter: Select • ?: Help • q: Quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		logo,
		subtitle,
		"\n",
		menu,
		help,
	)

	// Center the content
	box := BoxStyle.Width(70).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// renderHelp renders the help screen
func (m *MainMenuModel) renderHelp() string {
	helpContent := TitleStyle.Render("Keyboard Shortcuts") + "\n\n" +
		FormatKeyValue("↑ / k", "Move up") + "\n" +
		FormatKeyValue("↓ / j", "Move down") + "\n" +
		FormatKeyValue("Enter / Space", "Select item") + "\n" +
		FormatKeyValue("?", "Toggle this help") + "\n" +
		FormatKeyValue("Esc", "Go back / Close help") + "\n" +
		FormatKeyValue("q / Ctrl+C", "Quit application") + "\n\n" +
		SubtitleStyle.Render("About gimage") + "\n\n" +
		"gimage is a powerful CLI tool for AI-powered image generation\n" +
		"and processing. It supports multiple AI models including:\n\n" +
		FormatList([]string{
			"Gemini 2.5 Flash ($0.039/image)",
			"Gemini 3 Pro (Native 4K, $0.134/image)",
			"Imagen 4 (Best quality, $0.04/image)",
			"AWS Bedrock Nova Canvas ($0.08/image)",
			"xAI Grok Imagine ($0.02/image)",
		}) + "\n" +
		HelpStyle.Render("Press Esc to close this help")

	box := FocusedBoxStyle.Width(70).Render(helpContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// GetCurrentConfig returns the current configuration status
// This will be implemented in Phase 6
func (m *MainMenuModel) GetCurrentConfig() string {
	// TODO: Implement configuration status display
	return ""
}
