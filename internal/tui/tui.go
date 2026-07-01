// Package tui provides Terminal User Interface for gimage.
package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Screen represents the different screens in the TUI
type Screen int

const (
	ScreenMainMenu Screen = iota
	ScreenGenerate
	ScreenProcess
	ScreenSettings
	ScreenHelp
)

// Model is the main TUI model that handles all screens
type Model struct {
	currentScreen Screen
	width         int
	height        int
	err           error

	// Screen models
	mainMenu     *MainMenuModel
	generateFlow *GenerateFlowModel
	processMenu  *ProcessMenuModel
	settings     *SettingsMenuModel
}

// NewModel creates a new TUI model
func NewModel() *Model {
	return &Model{
		currentScreen: ScreenMainMenu,
		mainMenu:      NewMainMenuModel(),
	}
}

// Init initializes the TUI
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keyboard shortcuts
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentScreen == ScreenMainMenu {
				// Only quit from main menu
				return m, tea.Quit
			}
			// From other screens, go back to main menu
			m.currentScreen = ScreenMainMenu
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case NavigateMsg:
		// Handle navigation between screens
		m.currentScreen = msg.Screen
		switch msg.Screen {
		case ScreenGenerate:
			if m.generateFlow == nil {
				m.generateFlow = NewGenerateFlowModel()
			}
			return m, m.generateFlow.Init()
		case ScreenProcess:
			if m.processMenu == nil {
				m.processMenu = NewProcessMenuModel()
			}
			return m, m.processMenu.Init()
		case ScreenSettings:
			if m.settings == nil {
				m.settings = NewSettingsMenuModel()
			}
			return m, m.settings.Init()
		}
		return m, nil

	case error:
		m.err = msg
		return m, nil
	}

	// Delegate to current screen's update
	switch m.currentScreen {
	case ScreenMainMenu:
		return m.updateMainMenu(msg)
	case ScreenGenerate:
		return m.updateGenerateFlow(msg)
	case ScreenProcess:
		return m.updateProcessMenu(msg)
	case ScreenSettings:
		return m.updateSettings(msg)
	default:
		return m, nil
	}
}

// updateMainMenu updates the main menu
func (m *Model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.mainMenu, cmd = m.mainMenu.Update(msg)
	return m, cmd
}

// updateGenerateFlow updates the generate flow
func (m *Model) updateGenerateFlow(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.generateFlow == nil {
		m.generateFlow = NewGenerateFlowModel()
	}
	var cmd tea.Cmd
	m.generateFlow, cmd = m.generateFlow.Update(msg)
	return m, cmd
}

// updateProcessMenu updates the process menu
func (m *Model) updateProcessMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.processMenu == nil {
		m.processMenu = NewProcessMenuModel()
	}
	var cmd tea.Cmd
	m.processMenu, cmd = m.processMenu.Update(msg)
	return m, cmd
}

// updateSettings updates the settings menu
func (m *Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.settings == nil {
		m.settings = NewSettingsMenuModel()
	}
	var cmd tea.Cmd
	m.settings, cmd = m.settings.Update(msg)
	return m, cmd
}

// View renders the TUI. In bubbletea v2 the alt-screen and mouse mode are set
// declaratively on the returned View (they were tea.NewProgram options in v1).
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// render produces the content string for the current screen.
func (m *Model) render() string {
	if m.err != nil {
		return m.renderError()
	}

	switch m.currentScreen {
	case ScreenMainMenu:
		return m.mainMenu.View()
	case ScreenGenerate:
		if m.generateFlow != nil {
			return m.generateFlow.View()
		}
		return "Loading..."
	case ScreenProcess:
		if m.processMenu != nil {
			return m.processMenu.View()
		}
		return "Loading..."
	case ScreenSettings:
		if m.settings != nil {
			return m.settings.View()
		}
		return "Loading..."
	default:
		return "Screen not implemented yet"
	}
}

// renderError renders an error screen
func (m *Model) renderError() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorError).
		Padding(1, 2).
		Width(60).
		Align(lipgloss.Center)

	content := ErrorStyle.Render("Error") + "\n\n" +
		m.err.Error() + "\n\n" +
		MutedStyle.Render("Press 'q' to return to main menu")

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		style.Render(content),
	)
}

// NavigateMsg is a message to navigate to a different screen
type NavigateMsg struct {
	Screen Screen
}

// Navigate creates a navigation command
func Navigate(screen Screen) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{Screen: screen}
	}
}

// Run starts the TUI
func Run() error {
	// Alt-screen and mouse mode are now declared on the model's View (v2).
	p := tea.NewProgram(NewModel())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
