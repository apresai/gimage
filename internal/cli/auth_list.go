package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/apresai/gimage/internal/generate"
	"github.com/spf13/cobra"
)

// authListCmd shows all providers and their authentication status
var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all providers and their authentication status",
	Long: `Shows all available model providers, their required credentials,
and current authentication status.

The output shows:
- Provider ID and name
- Required credentials
- Current auth status (✓ configured, ✗ missing)
- Source of credentials (env/config/both)
- Pricing information`,
	Example: `  # List all providers and auth status
  gimage auth list

  # Show only configured providers
  gimage auth list --configured

  # Show only providers needing setup
  gimage auth list --missing`,
	RunE: runAuthList,
}

var (
	showConfiguredOnly bool
	showMissingOnly    bool
	showDetailed       bool
)

func init() {
	authListCmd.Flags().BoolVar(&showConfiguredOnly, "configured", false, "Show only configured providers")
	authListCmd.Flags().BoolVar(&showMissingOnly, "missing", false, "Show only providers missing credentials")
	authListCmd.Flags().BoolVar(&showDetailed, "detailed", false, "Show detailed credential requirements")
}

func runAuthList(cmd *cobra.Command, args []string) error {
	registry := generate.GetProviderRegistry()
	statuses := registry.GetAuthStatus()

	// Filter based on flags
	var filtered []generate.AuthStatus
	for _, status := range statuses {
		if showConfiguredOnly && !status.Configured {
			continue
		}
		if showMissingOnly && status.Configured {
			continue
		}
		filtered = append(filtered, status)
	}

	if len(filtered) == 0 {
		if showConfiguredOnly {
			fmt.Println("No providers are configured yet.")
		} else if showMissingOnly {
			fmt.Println("All providers are configured!")
		} else {
			fmt.Println("No providers found.")
		}
		return nil
	}

	if showDetailed {
		return showDetailedList(filtered)
	}

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "STATUS\tPROVIDER\tNAME\tPRICING\tSOURCE\tMISSING")
	fmt.Fprintln(w, "------\t--------\t----\t-------\t------\t-------")

	// List each provider
	for _, status := range filtered {
		p := status.Provider

		// Status icon
		statusIcon := "✗"
		if status.Configured {
			statusIcon = "✓"
		}

		// Pricing
		pricing := formatPricing(p.Pricing)

		// Missing credentials
		missing := "-"
		if len(status.Missing) > 0 {
			missing = strings.Join(status.Missing, ", ")
		}

		// Source
		source := status.Source
		if source == "none" {
			source = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			statusIcon,
			p.ID,
			p.Name,
			pricing,
			source,
			missing,
		)
	}

	// Footer with summary
	fmt.Println("\n" + strings.Repeat("─", 80))
	configured := 0
	for _, s := range statuses {
		if s.Configured {
			configured++
		}
	}
	fmt.Printf("Configured: %d/%d providers\n", configured, len(statuses))
	fmt.Println("\nUse 'gimage auth setup <provider>' to configure missing credentials")
	fmt.Println("Use 'gimage auth test <provider>' to test authentication")

	return nil
}

func showDetailedList(statuses []generate.AuthStatus) error {
	for i, status := range statuses {
		if i > 0 {
			fmt.Println("\n" + strings.Repeat("─", 80))
		}

		p := status.Provider

		// Header with status
		statusIcon := "✗ NOT CONFIGURED"
		statusColor := "\033[31m" // Red
		if status.Configured {
			statusIcon = "✓ CONFIGURED"
			statusColor = "\033[32m" // Green
		}

		fmt.Printf("%s%s\033[0m\n", statusColor, statusIcon)
		fmt.Printf("Provider ID: %s\n", p.ID)
		fmt.Printf("Name: %s\n", p.Name)
		fmt.Printf("Description: %s\n", p.Description)
		fmt.Printf("API: %s\n", p.API)
		fmt.Printf("Model: %s\n", p.ModelID)

		// Pricing
		fmt.Printf("\nPricing:\n")
		if p.Pricing.FreeTier {
			fmt.Printf("  • FREE tier: %s\n", p.Pricing.FreeTierLimit)
		}
		if p.Pricing.CostPerImage != nil {
			fmt.Printf("  • Cost per image: $%.4f %s\n", *p.Pricing.CostPerImage, p.Pricing.Currency)
		}

		// Required credentials
		fmt.Printf("\nRequired Credentials:\n")
		for _, env := range p.RequiredEnvVars {
			if !env.Required {
				continue
			}
			checkmark := "✗"
			value := os.Getenv(env.Name)
			if value != "" {
				checkmark = "✓"
				if env.Secret {
					value = maskSecret(value)
				}
			}
			fmt.Printf("  %s %s: %s\n", checkmark, env.Name, value)
			if env.Description != "" {
				fmt.Printf("      %s\n", env.Description)
			}
		}

		// Optional credentials
		hasOptional := false
		for _, env := range p.RequiredEnvVars {
			if !env.Required {
				hasOptional = true
				break
			}
		}
		if hasOptional {
			fmt.Printf("\nOptional Credentials:\n")
			for _, env := range p.RequiredEnvVars {
				if env.Required {
					continue
				}
				value := os.Getenv(env.Name)
				if value != "" && env.Secret {
					value = maskSecret(value)
				}
				if value == "" {
					value = "(not set)"
				}
				fmt.Printf("  • %s: %s\n", env.Name, value)
				if env.Description != "" {
					fmt.Printf("      %s\n", env.Description)
				}
			}
		}

		// Capabilities
		fmt.Printf("\nCapabilities:\n")
		caps := p.Capabilities
		fmt.Printf("  • Styles: %v\n", caps.SupportsStyles)
		fmt.Printf("  • Seed: %v\n", caps.SupportsSeed)
		fmt.Printf("  • Image size (1K/2K/4K): %v\n", caps.SupportsImageSize)
		fmt.Printf("  • Aspect ratio: %v\n", caps.SupportsAspectRatio)
		fmt.Printf("  • Max prompt length: %d tokens\n", caps.MaxPromptLength)
	}

	return nil
}

func formatPricing(p generate.PricingInfo) string {
	if p.FreeTier {
		return fmt.Sprintf("FREE (%s)", p.FreeTierLimit)
	}
	if p.CostPerImage != nil {
		return fmt.Sprintf("$%.4f/image", *p.CostPerImage)
	}
	return "Variable"
}

func maskSecret(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "..." + value[len(value)-4:]
}
