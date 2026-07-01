package cli

import (
	"fmt"
	"os"

	"github.com/apresai/gimage-deploy/internal/tui"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	Long: `Launch the interactive terminal UI for managing deployments and API keys.

The TUI provides a visual interface for:
- Managing Lambda deployments
- Creating and managing API keys
- Monitoring deployments
- Viewing logs and metrics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create TUI model
		model := tui.NewModel()

		// Create program (alt-screen is now declared on the model's View in v2)
		p := tea.NewProgram(model)

		// Run program
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			return err
		}

		return nil
	},
}
