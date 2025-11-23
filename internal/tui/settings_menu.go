// Package tui provides Terminal User Interface components for gimage.
package tui

import (
	"github.com/apresai/gimage/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SettingsPage represents different settings pages
type SettingsPage int

const (
	SettingsPageMenu SettingsPage = iota
	SettingsPageConfig
	SettingsPageAPIStatus
	SettingsPageAbout
	SettingsPageShortcuts
)

// SettingsMenuModel handles settings and configuration
type SettingsMenuModel struct {
	width       int
	height      int
	showHelp    bool
	cfg         *config.Config
	selectedOp  int
	options     []string
	currentPage SettingsPage

	// Editing state
	editingKey   string // Which key is being edited ("gemini", "vertex", "aws", etc.)
	editInput    textinput.Model
	saveMessage  string
	saveError    error
}

// NewSettingsMenuModel creates a new settings menu model
func NewSettingsMenuModel() *SettingsMenuModel {
	// Try to load config
	cfg, _ := config.LoadConfig()

	// Initialize text input for editing
	editInput := textinput.New()
	editInput.Placeholder = "Enter API key..."
	editInput.CharLimit = 256
	editInput.Width = 60

	return &SettingsMenuModel{
		cfg:         cfg,
		currentPage: SettingsPageMenu,
		editInput:   editInput,
		options: []string{
			"View Configuration",
			"Check API Keys Status",
			"About gimage",
			"Keyboard Shortcuts",
		},
	}
}

// Init initializes the settings menu
func (m *SettingsMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the settings menu
func (m *SettingsMenuModel) Update(msg tea.Msg) (*SettingsMenuModel, tea.Cmd) {
	var cmd tea.Cmd

	// If in editing mode, handle input
	if m.editingKey != "" {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Save the edited value
				return m, m.saveAPIKey()
			case "esc":
				// Cancel editing
				m.editingKey = ""
				m.saveMessage = ""
				m.saveError = nil
				m.editInput.Blur()
				return m, nil
			}
		}
		// Update the input
		m.editInput, cmd = m.editInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc", "m":
			// If on a sub-page, go back to menu
			if m.currentPage != SettingsPageMenu {
				m.currentPage = SettingsPageMenu
				m.saveMessage = ""
				m.saveError = nil
				return m, nil
			}
			// If on menu, go to main menu
			return m, Navigate(ScreenMainMenu)
		case "up", "k":
			if m.selectedOp > 0 {
				m.selectedOp--
			}
		case "down", "j":
			if m.selectedOp < len(m.options)-1 {
				m.selectedOp++
			}
		case "enter", " ":
			// Only handle enter on the menu page
			if m.currentPage == SettingsPageMenu {
				switch m.selectedOp {
				case 0:
					m.currentPage = SettingsPageConfig
				case 1:
					m.currentPage = SettingsPageAPIStatus
				case 2:
					m.currentPage = SettingsPageAbout
				case 3:
					m.currentPage = SettingsPageShortcuts
				}
			}
			return m, nil
		case "e":
			// Edit API keys on config page
			if m.currentPage == SettingsPageConfig {
				return m, m.startEditingAPIKey()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the settings menu
func (m *SettingsMenuModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	switch m.currentPage {
	case SettingsPageMenu:
		return m.viewMainSettings()
	case SettingsPageConfig:
		return m.viewConfig()
	case SettingsPageAPIStatus:
		return m.viewAPIStatus()
	case SettingsPageAbout:
		return m.viewAbout()
	case SettingsPageShortcuts:
		return m.viewShortcuts()
	default:
		return m.viewMainSettings()
	}
}

func (m *SettingsMenuModel) viewMainSettings() string {
	var items []string
	for i, opt := range m.options {
		cursor := "  "
		style := MenuItemStyle
		if i == m.selectedOp {
			cursor = "> "
			style = SelectedMenuItemStyle
		}
		items = append(items, style.Render(cursor+opt))
	}

	content := TitleStyle.Render("Settings") + "\n\n" +
		SubtitleStyle.Render("Configuration & Information") + "\n\n" +
		lipgloss.JoinVertical(lipgloss.Left, items...) + "\n\n" +
		HelpStyle.Render("↑/↓: Navigate • Enter: Select • m: Main menu • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *SettingsMenuModel) viewConfig() string {
	// If in editing mode, show edit interface
	if m.editingKey != "" {
		return m.viewEditAPIKey()
	}

	var configLines []string

	if m.cfg != nil {
		configLines = []string{
			FormatKeyValue("Config File", "~/.gimage/config.md"),
			"",
			SubtitleStyle.Render("Current Configuration"),
			"",
		}

		if m.cfg.GeminiAPIKey != "" {
			configLines = append(configLines, FormatKeyValue("Gemini API Key", "Set ("+maskKey(m.cfg.GeminiAPIKey)+")"))
		} else {
			configLines = append(configLines, FormatKeyValue("Gemini API Key", ErrorStyle.Render("Not set")))
		}

		if m.cfg.VertexAPIKey != "" {
			configLines = append(configLines, FormatKeyValue("Vertex API Key", "Set ("+maskKey(m.cfg.VertexAPIKey)+")"))
			configLines = append(configLines, FormatKeyValue("Vertex Project", m.cfg.VertexProject))
		} else {
			configLines = append(configLines, FormatKeyValue("Vertex API Key", ErrorStyle.Render("Not set")))
		}

		if m.cfg.AWSAccessKeyID != "" {
			configLines = append(configLines, FormatKeyValue("AWS Access Key", "Set ("+maskKey(m.cfg.AWSAccessKeyID)+")"))
			configLines = append(configLines, FormatKeyValue("AWS Region", m.cfg.AWSRegion))
		} else {
			configLines = append(configLines, FormatKeyValue("AWS Access Key", ErrorStyle.Render("Not set")))
		}

		configLines = append(configLines, "", FormatKeyValue("Default API", m.cfg.DefaultAPI))
		configLines = append(configLines, FormatKeyValue("Default Model", m.cfg.DefaultModel))
	} else {
		configLines = []string{
			ErrorStyle.Render("Configuration file not found"),
			"",
			MutedStyle.Render("Run 'gimage auth gemini' to set up credentials"),
		}
	}

	breadcrumb := MutedStyle.Render("Settings > View Configuration")
	content := breadcrumb + "\n\n" +
		TitleStyle.Render("Configuration") + "\n\n" +
		lipgloss.JoinVertical(lipgloss.Left, configLines...) + "\n\n"

	// Show save message if exists
	if m.saveMessage != "" {
		if m.saveError != nil {
			content += ErrorStyle.Render("Error: "+m.saveError.Error()) + "\n\n"
		} else {
			content += SuccessStyle.Render(m.saveMessage) + "\n\n"
		}
	}

	content += WarningStyle.Render("Press 'e' to edit API keys") + "\n\n" +
		HelpStyle.Render("e: Edit keys • Esc: Back to settings menu • m: Main menu")

	box := FocusedBoxStyle.Width(80).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *SettingsMenuModel) viewAPIStatus() string {
	hasGemini := config.HasGeminiCredentials()
	hasVertex := config.HasVertexCredentials()
	hasBedrock := config.HasBedrockCredentials()

	geminiStatus := redNo()
	vertexStatus := redNo()
	bedrockStatus := redNo()

	if hasGemini {
		geminiStatus = greenYes()
	}
	if hasVertex {
		vertexStatus = greenYes()
	}
	if hasBedrock {
		bedrockStatus = greenYes()
	}

	breadcrumb := MutedStyle.Render("Settings > Check API Keys Status")
	content := breadcrumb + "\n\n" +
		TitleStyle.Render("API Keys Status") + "\n\n" +
		SubtitleStyle.Render("Authentication Status") + "\n\n" +
		FormatKeyValue("Gemini API", geminiStatus+" "+(map[bool]string{true: "Configured", false: "Not configured"}[hasGemini])) + "\n" +
		MutedStyle.Render("  Free tier: 500 images/day") + "\n\n" +
		FormatKeyValue("Vertex AI", vertexStatus+" "+(map[bool]string{true: "Configured", false: "Not configured"}[hasVertex])) + "\n" +
		MutedStyle.Render("  Paid: $0.02-0.06 per image") + "\n\n" +
		FormatKeyValue("AWS Bedrock", bedrockStatus+" "+(map[bool]string{true: "Configured", false: "Not configured"}[hasBedrock])) + "\n" +
		MutedStyle.Render("  Paid: $0.04-0.08 per image") + "\n\n" +
		WarningStyle.Render("Setup: gimage auth <api>") + "\n\n" +
		HelpStyle.Render("Esc: Back to settings menu • m: Main menu")

	box := FocusedBoxStyle.Width(76).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *SettingsMenuModel) viewAbout() string {
	breadcrumb := MutedStyle.Render("Settings > About gimage")
	content := breadcrumb + "\n\n" +
		TitleStyle.Render("About gimage") + "\n\n" +
		SubtitleStyle.Render("AI Image Generation & Processing Tool") + "\n\n" +
		"gimage is a powerful CLI and TUI for generating\n" +
		"AI images and processing existing images.\n\n" +
		SubtitleStyle.Render("Supported AI Models") + "\n\n" +
		"• Gemini 2.5 Flash (Free tier)\n" +
		"• Imagen 4 (Premium quality)\n" +
		"• AWS Bedrock Nova Canvas\n\n" +
		SubtitleStyle.Render("Image Processing") + "\n\n" +
		"• Resize, Scale, Crop\n" +
		"• Compress (JPEG quality)\n" +
		"• Convert (PNG, JPG, WebP, etc)\n\n" +
		MutedStyle.Render("Built with Go • Pure Go (zero C dependencies)") + "\n\n" +
		HelpStyle.Render("Esc: Back to settings menu • m: Main menu")

	box := FocusedBoxStyle.Width(76).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *SettingsMenuModel) viewShortcuts() string {
	breadcrumb := MutedStyle.Render("Settings > Keyboard Shortcuts")
	content := breadcrumb + "\n\n" +
		TitleStyle.Render("Keyboard Shortcuts") + "\n\n" +
		SubtitleStyle.Render("Global Shortcuts") + "\n\n" +
		FormatKeyValue("Ctrl+C, q", "Quit application") + "\n" +
		FormatKeyValue("Esc", "Go back / Cancel") + "\n" +
		FormatKeyValue("?", "Toggle help") + "\n\n" +
		SubtitleStyle.Render("Navigation") + "\n\n" +
		FormatKeyValue("↑/↓ or k/j", "Move up/down") + "\n" +
		FormatKeyValue("Enter/Space", "Select item") + "\n" +
		FormatKeyValue("Tab", "Next input field") + "\n" +
		FormatKeyValue("Shift+Tab", "Previous input field") + "\n\n" +
		SubtitleStyle.Render("Special Keys") + "\n\n" +
		FormatKeyValue("Enter", "Submit prompt (generate)") + "\n" +
		FormatKeyValue("Shift+Enter", "New line in prompt") + "\n" +
		FormatKeyValue("Ctrl+L", "Clear screen") + "\n" +
		FormatKeyValue("m", "Return to main menu") + "\n\n" +
		HelpStyle.Render("Esc: Back to settings menu • m: Main menu")

	box := FocusedBoxStyle.Width(76).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *SettingsMenuModel) renderHelp() string {
	helpContent := TitleStyle.Render("Settings Help") + "\n\n" +
		"View and manage gimage configuration." + "\n\n" +
		SubtitleStyle.Render("Configuration Location") + "\n\n" +
		"Config file: ~/.gimage/config.md\n" +
		"Format: Markdown with key-value pairs\n\n" +
		SubtitleStyle.Render("Setting Up API Keys") + "\n\n" +
		"Use the CLI commands:\n" +
		"  gimage auth gemini\n" +
		"  gimage auth vertex\n" +
		"  gimage auth bedrock\n\n" +
		HelpStyle.Render("Press Esc to close")

	box := FocusedBoxStyle.Width(70).Render(helpContent)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func greenYes() string {
	return SuccessStyle.Render("✓")
}

func redNo() string {
	return ErrorStyle.Render("✗")
}

// startEditingAPIKey prompts the user to select which API key to edit
func (m *SettingsMenuModel) startEditingAPIKey() tea.Cmd {
	// For simplicity, let's just edit Gemini API key first
	// In a more complete implementation, we'd show a menu to select which key
	m.editingKey = "gemini"
	m.editInput.SetValue("")
	if m.cfg != nil && m.cfg.GeminiAPIKey != "" {
		m.editInput.SetValue(m.cfg.GeminiAPIKey)
	}
	m.editInput.Focus()
	m.saveMessage = ""
	m.saveError = nil
	return textinput.Blink
}

// viewEditAPIKey renders the API key editing interface
func (m *SettingsMenuModel) viewEditAPIKey() string {
	var keyName string
	switch m.editingKey {
	case "gemini":
		keyName = "Gemini API Key"
	case "vertex":
		keyName = "Vertex API Key"
	case "aws":
		keyName = "AWS Access Key"
	default:
		keyName = "API Key"
	}

	breadcrumb := MutedStyle.Render("Settings > View Configuration > Edit API Key")
	content := breadcrumb + "\n\n" +
		TitleStyle.Render("Edit "+keyName) + "\n\n" +
		SubtitleStyle.Render("Enter your API key") + "\n\n" +
		m.editInput.View() + "\n\n" +
		MutedStyle.Render("Your key will be saved to ~/.gimage/config.md") + "\n" +
		MutedStyle.Render("The file is secured with 0600 permissions") + "\n\n" +
		HelpStyle.Render("Enter: Save • Esc: Cancel")

	box := FocusedBoxStyle.Width(80).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// saveAPIKey saves the edited API key to config
func (m *SettingsMenuModel) saveAPIKey() tea.Cmd {
	return func() tea.Msg {
		newValue := m.editInput.Value()

		// Reload config to ensure we have latest values
		cfg, err := config.LoadConfig()
		if err != nil {
			// If no config exists, create a new one
			cfg = &config.Config{
				DefaultAPI:   "gemini",
				DefaultModel: "gemini-3-pro-image-preview",
				DefaultSize:  "1024x1024",
			}
		}

		// Update the appropriate field
		switch m.editingKey {
		case "gemini":
			cfg.GeminiAPIKey = newValue
		case "vertex":
			cfg.VertexAPIKey = newValue
		case "aws":
			cfg.AWSAccessKeyID = newValue
		}

		// Save config
		if err := config.SaveConfig(cfg); err != nil {
			m.saveError = err
			m.saveMessage = ""
			m.editingKey = ""
			m.editInput.Blur()
			return nil
		}

		// Update local config
		m.cfg = cfg
		m.saveMessage = "✓ API key saved successfully"
		m.saveError = nil
		m.editingKey = ""
		m.editInput.Blur()

		return nil
	}
}
