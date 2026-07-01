package tui

import (
	"github.com/apresai/gimage-deploy/internal/storage"
	tea "charm.land/bubbletea/v2"
)

// Screen represents different TUI screens
type Screen int

const (
	ScreenMainMenu Screen = iota
	ScreenDeploymentList
	ScreenDeploymentDetail
	ScreenAPIKeyList
	ScreenAPIKeyDetail
	ScreenSettings
)

// Model is the main TUI model
type Model struct {
	screen        Screen
	width         int
	height        int
	deploymentMgr *storage.DeploymentManager
	keyMgr        *storage.APIKeyManager
	configMgr     *storage.ConfigManager

	// Sub-models for different screens
	mainMenu       *MainMenuModel
	deploymentList *DeploymentListModel
	apiKeyList     *APIKeyListModel
}

// NewModel creates a new TUI model
func NewModel() Model {
	dm := storage.NewDeploymentManager()
	dm.Load()

	km := storage.NewAPIKeyManager()
	km.Load()

	cm := storage.NewConfigManager()
	cm.Load()

	return Model{
		screen:         ScreenMainMenu,
		deploymentMgr:  dm,
		keyMgr:         km,
		configMgr:      cm,
		mainMenu:       NewMainMenuModel(),
		deploymentList: NewDeploymentListModel(dm),
		apiKeyList:     NewAPIKeyListModel(km),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == ScreenMainMenu {
				return m, tea.Quit
			}
			// Go back to main menu
			m.screen = ScreenMainMenu
			return m, nil
		case "esc":
			// Go back to previous screen
			if m.screen != ScreenMainMenu {
				m.screen = ScreenMainMenu
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Delegate to appropriate sub-model
	switch m.screen {
	case ScreenMainMenu:
		updated, cmd := m.mainMenu.Update(msg)
		m.mainMenu = &updated
		// Check for screen change
		if m.mainMenu.selectedScreen != ScreenMainMenu {
			m.screen = m.mainMenu.selectedScreen
			m.mainMenu.selectedScreen = ScreenMainMenu // Reset
		}
		return m, cmd

	case ScreenDeploymentList:
		updated, cmd := m.deploymentList.Update(msg)
		m.deploymentList = &updated
		return m, cmd

	case ScreenAPIKeyList:
		updated, cmd := m.apiKeyList.Update(msg)
		m.apiKeyList = &updated
		return m, cmd
	}

	return m, nil
}

// View renders the model. In bubbletea v2 the alt-screen is declared on the
// returned View instead of via a tea.NewProgram option.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render produces the content string for the current screen.
func (m Model) render() string {
	switch m.screen {
	case ScreenMainMenu:
		return m.mainMenu.View()
	case ScreenDeploymentList:
		return m.deploymentList.View()
	case ScreenAPIKeyList:
		return m.apiKeyList.View()
	default:
		return "Unknown screen"
	}
}
