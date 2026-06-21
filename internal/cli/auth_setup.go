package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// authSetupCmd interactively sets up authentication for a provider
var authSetupCmd = &cobra.Command{
	Use:   "setup <provider>",
	Short: "Set up authentication for a specific provider",
	Long: `Interactive setup wizard for configuring provider authentication.

This command will:
1. Show what credentials are required
2. Guide you through obtaining them
3. Test the configuration
4. Save to config file`,
	Example: `  # Set up Gemini authentication
  gimage auth setup gemini/flash-2.5

  # Set up using alias
  gimage auth setup gemini

  # Non-interactive with flags
  gimage auth setup gemini --api-key YOUR_KEY`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthSetup,
}

func init() {
	// Provider-specific flags will be added dynamically
}

func runAuthSetup(cmd *cobra.Command, args []string) error {
	registry := generate.GetProviderRegistry()

	// Resolve provider
	provider, err := registry.ResolveProvider(args[0])
	if err != nil {
		return fmt.Errorf("unknown provider: %s", args[0])
	}

	fmt.Printf("Setting up: %s\n", provider.Name)
	fmt.Printf("Provider ID: %s\n", provider.ID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	// Show description
	fmt.Println(provider.Description)
	fmt.Println()

	// Show pricing
	fmt.Println("Pricing:")
	if provider.Pricing.FreeTier {
		fmt.Printf("  • FREE tier: %s\n", provider.Pricing.FreeTierLimit)
	}
	if provider.Pricing.CostPerImage != nil {
		fmt.Printf("  • Cost per image: $%.4f\n", *provider.Pricing.CostPerImage)
	}
	fmt.Println()

	// Load existing config
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = &config.Config{}
	}

	// Collect credentials
	reader := bufio.NewReader(os.Stdin)
	credentials := make(map[string]string)

	fmt.Println("Required credentials:")
	fmt.Println(strings.Repeat("─", 60))

	for _, env := range provider.RequiredEnvVars {
		if !env.Required {
			continue
		}

		fmt.Printf("\n%s:\n", env.Name)
		if env.Description != "" {
			fmt.Printf("  %s\n", env.Description)
		}

		// Check if already set
		existingValue := getExistingValue(env, cfg)
		if existingValue != "" {
			if env.Secret {
				fmt.Printf("  Current value: %s\n", maskSecret(existingValue))
			} else {
				fmt.Printf("  Current value: %s\n", existingValue)
			}
			fmt.Print("  Keep existing value? (Y/n): ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "n" && answer != "no" {
				credentials[env.Name] = existingValue
				continue
			}
		}

		// Get new value
		var value string
		if env.Secret {
			fmt.Printf("  Enter %s: ", env.Name)
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			value = string(bytePassword)
		} else {
			fmt.Printf("  Enter %s: ", env.Name)
			value, _ = reader.ReadString('\n')
			value = strings.TrimSpace(value)
		}

		if value == "" {
			return fmt.Errorf("%s is required", env.Name)
		}

		credentials[env.Name] = value
	}

	// Handle optional credentials
	fmt.Println("\nOptional credentials (press Enter to skip):")
	fmt.Println(strings.Repeat("─", 60))

	for _, env := range provider.RequiredEnvVars {
		if env.Required {
			continue
		}

		fmt.Printf("\n%s (optional):\n", env.Name)
		if env.Description != "" {
			fmt.Printf("  %s\n", env.Description)
		}

		existingValue := getExistingValue(env, cfg)
		if existingValue != "" {
			if env.Secret {
				fmt.Printf("  Current value: %s\n", maskSecret(existingValue))
			} else {
				fmt.Printf("  Current value: %s\n", existingValue)
			}
			fmt.Print("  Keep existing value? (Y/n): ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "n" && answer != "no" {
				credentials[env.Name] = existingValue
				continue
			}
		}

		var value string
		if env.Secret {
			fmt.Printf("  Enter %s (optional): ", env.Name)
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err == nil {
				value = string(bytePassword)
			}
		} else {
			fmt.Printf("  Enter %s (optional): ", env.Name)
			value, _ = reader.ReadString('\n')
			value = strings.TrimSpace(value)
		}

		if value != "" {
			credentials[env.Name] = value
		}
	}

	// Test the configuration
	fmt.Println("\nTesting configuration...")
	fmt.Println(strings.Repeat("─", 60))

	// Create a temporary provider with test credentials
	testClient, err := provider.CreateClient(credentials)
	if err != nil {
		fmt.Printf("✗ Failed to create client: %v\n", err)
		fmt.Print("\nSave configuration anyway? (y/N): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			return fmt.Errorf("setup cancelled")
		}
	} else {
		testClient.Close()
		fmt.Println("✓ Client created successfully")
	}

	// Save to config
	fmt.Print("\nSave to config file? (Y/n): ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "n" && answer != "no" {
		// Update config with new values
		for _, env := range provider.RequiredEnvVars {
			value := credentials[env.Name]
			if value == "" {
				continue
			}

			switch env.ConfigKey {
			case "gemini_api_key":
				cfg.GeminiAPIKey = value
			case "vertex_project":
				cfg.VertexProject = value
			case "vertex_location":
				cfg.VertexLocation = value
			case "vertex_api_key":
				cfg.VertexAPIKey = value
			}
		}

		// Save config
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("✓ Configuration saved to ~/.gimage/config.md")
	}

	fmt.Println("\n✓ Setup complete!")
	fmt.Printf("\nYou can now use: gimage generate \"your prompt\" --provider %s\n", provider.ID)

	return nil
}

func getExistingValue(env generate.EnvVar, cfg *config.Config) string {
	// Check environment first
	if val := os.Getenv(env.Name); val != "" {
		return val
	}

	// Check config
	switch env.ConfigKey {
	case "gemini_api_key":
		return cfg.GeminiAPIKey
	case "vertex_project":
		return cfg.VertexProject
	case "vertex_location":
		return cfg.VertexLocation
	case "vertex_api_key":
		return cfg.VertexAPIKey
	}

	return ""
}
