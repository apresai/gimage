package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginLeft(2)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1).
			MarginLeft(2)
)

type menuItem struct {
	title       string
	description string
	screen      Screen
}

// MainMenuModel represents the main menu
type MainMenuModel struct {
	cursor         int
	items          []menuItem
	selectedScreen Screen
}

// NewMainMenuModel creates a new main menu
func NewMainMenuModel() *MainMenuModel {
	return &MainMenuModel{
		cursor: 0,
		items: []menuItem{
			{"Deployments", "Manage Lambda deployments", ScreenDeploymentList},
			{"API Keys", "Manage API Gateway keys", ScreenAPIKeyList},
			{"Settings", "Configure defaults", ScreenSettings},
			{"Quit", "Exit application", ScreenMainMenu},
		},
		selectedScreen: ScreenMainMenu,
	}
}

// Update handles messages for the main menu
func (m MainMenuModel) Update(msg tea.Msg) (MainMenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			selected := m.items[m.cursor]
			if selected.title == "Quit" {
				return m, tea.Quit
			}
			m.selectedScreen = selected.screen
			return m, nil
		}
	}
	return m, nil
}

// View renders the main menu
func (m MainMenuModel) View() string {
	s := titleStyle.Render("GIMAGE DEPLOY CLI")
	s += "\n"
	s += titleStyle.Render("Lambda Deployment & API Key Management")
	s += "\n\n"

	for i, item := range m.items {
		cursor := " "
		style := menuItemStyle

		if m.cursor == i {
			cursor = ">"
			style = selectedItemStyle
		}

		s += style.Render(fmt.Sprintf("%s %s", cursor, item.title))
		s += "\n"
		if m.cursor == i {
			s += helpStyle.Render(fmt.Sprintf("   %s", item.description))
			s += "\n"
		}
	}

	s += "\n"
	s += helpStyle.Render("↑/↓: Navigate • Enter: Select • q: Quit")

	return s
}
