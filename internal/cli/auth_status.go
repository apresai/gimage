package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/apresai/gimage/internal/config"
	"github.com/spf13/cobra"
)

// authStatusCmd shows authentication status for all providers
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status for all providers",
	Long: `Display authentication status for all configured providers, showing:
  • Which credentials are configured
  • Where credentials are coming from (flags, env vars, config file)
  • Warnings about conflicting credentials
  • Masked preview of API keys

Examples:
  gimage auth status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printAuthStatus()
		return nil
	},
}

func init() {
	authCmd.AddCommand(authStatusCmd)
}

func printAuthStatus() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Authentication Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Credential Priority: CLI Flags > Environment Variables > Config File > Defaults")
	fmt.Println()

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("⚠ Warning: Could not load config file: %v\n", err)
		cfg = &config.Config{} // Use empty config
	}

	// Check Gemini API
	printProviderAuthStatus("Gemini API", checkGeminiAuth(cfg))
	fmt.Println()

	// Check Vertex AI
	printProviderAuthStatus("Vertex AI", checkVertexAuth(cfg))
	fmt.Println()

	// Show credential priority diagram
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Credential Priority Hierarchy")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("    1. CLI Flags       (highest priority)")
	fmt.Println("        ↓")
	fmt.Println("    2. Environment Variables")
	fmt.Println("        ↓")
	fmt.Println("    3. Config File (~/.gimage/config.md)")
	fmt.Println("        ↓")
	fmt.Println("    4. Defaults        (lowest priority)")
	fmt.Println()
}

type authStatus struct {
	configured bool
	sources    []string
	conflicts  []string
	key        string
}

func checkGeminiAuth(cfg *config.Config) authStatus {
	status := authStatus{sources: []string{}, conflicts: []string{}}

	// Check environment variable
	envKey := os.Getenv("GEMINI_API_KEY")
	if envKey != "" {
		status.configured = true
		status.sources = append(status.sources, fmt.Sprintf("Environment: GEMINI_API_KEY = %s", maskKey(envKey)))
		status.key = envKey
	}

	// Check config file
	if cfg.GeminiAPIKey != "" {
		if status.configured {
			status.conflicts = append(status.conflicts, "Also found in config file (environment takes precedence)")
		} else {
			status.configured = true
			status.sources = append(status.sources, fmt.Sprintf("Config file: gemini_api_key = %s", maskKey(cfg.GeminiAPIKey)))
			status.key = cfg.GeminiAPIKey
		}
	}

	return status
}

func checkVertexAuth(cfg *config.Config) authStatus {
	status := authStatus{sources: []string{}, conflicts: []string{}}

	// Check for API key (Express Mode)
	envAPIKey := os.Getenv("VERTEX_API_KEY")
	if envAPIKey != "" {
		status.configured = true
		status.sources = append(status.sources, fmt.Sprintf("Environment: VERTEX_API_KEY = %s (Express Mode)", maskKey(envAPIKey)))
	}

	if cfg.VertexAPIKey != "" {
		if envAPIKey != "" {
			status.conflicts = append(status.conflicts, "Also found in config file (environment takes precedence)")
		} else {
			status.configured = true
			status.sources = append(status.sources, fmt.Sprintf("Config file: vertex_api_key = %s (Express Mode)", maskKey(cfg.VertexAPIKey)))
		}
	}

	// Check for project/location
	envProject := os.Getenv("VERTEX_PROJECT")
	envLocation := os.Getenv("VERTEX_LOCATION")

	if envProject != "" {
		status.sources = append(status.sources, fmt.Sprintf("Environment: VERTEX_PROJECT = %s", envProject))
	} else if cfg.VertexProject != "" {
		status.sources = append(status.sources, fmt.Sprintf("Config file: vertex_project = %s", cfg.VertexProject))
	}

	if envLocation != "" {
		status.sources = append(status.sources, fmt.Sprintf("Environment: VERTEX_LOCATION = %s", envLocation))
	} else if cfg.VertexLocation != "" {
		status.sources = append(status.sources, fmt.Sprintf("Config file: vertex_location = %s", cfg.VertexLocation))
	}

	// Check for service account (Full Mode)
	googleCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if googleCreds != "" {
		status.configured = true
		status.sources = append(status.sources, fmt.Sprintf("Environment: GOOGLE_APPLICATION_CREDENTIALS = %s (Full Mode)", googleCreds))
	}

	return status
}

func printProviderAuthStatus(name string, status authStatus) {
	if status.configured {
		fmt.Printf("✓ %s (Configured)\n", name)
		for _, source := range status.sources {
			fmt.Printf("  • %s\n", source)
		}
		for _, conflict := range status.conflicts {
			fmt.Printf("  ⚠ %s\n", conflict)
		}
	} else {
		fmt.Printf("✗ %s (Not Configured)\n", name)
		fmt.Printf("  Run: gimage auth setup %s\n", strings.ToLower(strings.Split(name, " ")[0]))
	}
}

func maskKey(key string) string {
	if key == "" {
		return "<empty>"
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
