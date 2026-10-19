package cli

import (
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication for all image generation providers",
	Long: `Manage authentication credentials for all supported providers.

Each provider (Gemini API, Vertex AI, xAI Grok) has specific credential
requirements and pricing. Use the subcommands to:

  list  - View all providers and their authentication status
  test  - Test if authentication works for a provider
  setup - Interactively configure credentials for a provider

Different providers offer different models with different pricing:
  - gemini/nano-banana-2.1: $0.0336/image (1K) via Gemini API (default)
  - gemini/flash-3.1-lite: $0.034/image via Gemini API (1K only)
  - vertex/nano-banana-2.1: Gemini Nano Banana 2.1 tiered pricing via Vertex AI
  - grok/grok-imagine-2.0: $0.04/image (1K), $0.06 (2K) via xAI Grok`,
	Example: `  # List all providers and auth status
  gimage auth list

  # Set up Gemini authentication
  gimage auth setup gemini

  # Test if Vertex AI works
  gimage auth test vertex/nano-banana-2.1`,
}

func init() {
	rootCmd.AddCommand(authCmd)
	// Provider-based commands
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authTestCmd)
	authCmd.AddCommand(authSetupCmd)
}
