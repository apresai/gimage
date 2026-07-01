package tui

import (
	"fmt"

	"github.com/apresai/gimage-deploy/internal/models"
	"github.com/apresai/gimage-deploy/internal/storage"
	"github.com/apresai/gimage-deploy/pkg/utils"
	tea "charm.land/bubbletea/v2"
)

// APIKeyListModel represents the API key list view
type APIKeyListModel struct {
	cursor int
	keys   []*models.APIKey
	km     *storage.APIKeyManager
}

// NewAPIKeyListModel creates a new API key list model
func NewAPIKeyListModel(km *storage.APIKeyManager) *APIKeyListModel {
	return &APIKeyListModel{
		cursor: 0,
		keys:   km.List(),
		km:     km,
	}
}

// Update handles messages for the API key list
func (m APIKeyListModel) Update(msg tea.Msg) (APIKeyListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.keys)-1 {
				m.cursor++
			}
		case "r":
			// Refresh list
			m.km.Load()
			m.keys = m.km.List()
			if m.cursor >= len(m.keys) {
				m.cursor = len(m.keys) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		}
	}
	return m, nil
}

// View renders the API key list
func (m APIKeyListModel) View() string {
	s := titleStyle.Render("API KEYS")
	s += "\n\n"

	if len(m.keys) == 0 {
		s += helpStyle.Render("No API keys found.")
		s += "\n\n"
		s += helpStyle.Render("Create one with: gimage-deploy keys create --name <name> --deployment <id>")
		s += "\n"
		return s
	}

	// Table header
	header := tableHeaderStyle.Render(
		fmt.Sprintf("%-20s %-15s %-10s %-35s",
			"NAME", "DEPLOYMENT", "STATUS", "KEY VALUE"))
	s += header + "\n"

	// Table rows
	for i, key := range m.keys {
		maskedKey := utils.MaskAPIKey(key.KeyValue)

		statusStr := string(key.Status)
		statusStyle := tableRowStyle
		if key.Status == models.APIKeyActive {
			statusStyle = statusActiveStyle
		} else if key.Status == models.APIKeyDisabled {
			statusStyle = statusFailedStyle
		}

		row := fmt.Sprintf("%-20s %-15s %-10s %-35s",
			key.Name, key.DeploymentID, statusStyle.Render(statusStr), maskedKey)

		if i == m.cursor {
			s += selectedRowStyle.Render(row) + "\n"
		} else {
			s += tableRowStyle.Render(row) + "\n"
		}
	}

	s += "\n"
	s += helpStyle.Render(fmt.Sprintf("Showing %d API key(s)", len(m.keys)))
	s += "\n"
	s += helpStyle.Render("↑/↓: Navigate • r: Refresh • ESC: Back • q: Quit")

	return s
}
