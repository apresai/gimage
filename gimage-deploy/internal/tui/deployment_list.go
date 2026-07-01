package tui

import (
	"fmt"

	"github.com/apresai/gimage-deploy/internal/models"
	"github.com/apresai/gimage-deploy/internal/storage"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color("240"))

	tableRowStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

	selectedRowStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				PaddingRight(1).
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("170"))

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("Green"))

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("Red"))
)

// DeploymentListModel represents the deployment list view
type DeploymentListModel struct {
	cursor      int
	deployments []*models.Deployment
	dm          *storage.DeploymentManager
}

// NewDeploymentListModel creates a new deployment list model
func NewDeploymentListModel(dm *storage.DeploymentManager) *DeploymentListModel {
	return &DeploymentListModel{
		cursor:      0,
		deployments: dm.List(),
		dm:          dm,
	}
}

// Update handles messages for the deployment list
func (m DeploymentListModel) Update(msg tea.Msg) (DeploymentListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.deployments)-1 {
				m.cursor++
			}
		case "r":
			// Refresh list
			m.dm.Load()
			m.deployments = m.dm.List()
			if m.cursor >= len(m.deployments) {
				m.cursor = len(m.deployments) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		}
	}
	return m, nil
}

// View renders the deployment list
func (m DeploymentListModel) View() string {
	s := titleStyle.Render("DEPLOYMENTS")
	s += "\n\n"

	if len(m.deployments) == 0 {
		s += helpStyle.Render("No deployments found.")
		s += "\n\n"
		s += helpStyle.Render("Create one with: gimage-deploy deploy --id <id> --stage <stage>")
		s += "\n"
		return s
	}

	// Table header
	header := tableHeaderStyle.Render(
		fmt.Sprintf("%-15s %-12s %-10s %-10s %-40s",
			"ID", "REGION", "STAGE", "STATUS", "ENDPOINT"))
	s += header + "\n"

	// Table rows
	for i, d := range m.deployments {
		endpoint := d.APIGatewayURL
		if len(endpoint) > 40 {
			endpoint = endpoint[:37] + "..."
		}

		statusStr := string(d.Status)
		statusStyle := tableRowStyle
		if d.Status == models.StatusActive {
			statusStyle = statusActiveStyle
		} else if d.Status == models.StatusFailed {
			statusStyle = statusFailedStyle
		}

		row := fmt.Sprintf("%-15s %-12s %-10s %-10s %-40s",
			d.ID, d.Region, d.Stage, statusStyle.Render(statusStr), endpoint)

		if i == m.cursor {
			s += selectedRowStyle.Render(row) + "\n"
		} else {
			s += tableRowStyle.Render(row) + "\n"
		}
	}

	s += "\n"
	s += helpStyle.Render(fmt.Sprintf("Showing %d deployment(s)", len(m.deployments)))
	s += "\n"
	s += helpStyle.Render("↑/↓: Navigate • r: Refresh • ESC: Back • q: Quit")

	return s
}
