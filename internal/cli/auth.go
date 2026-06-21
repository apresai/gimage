package cli

import (
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication for all image generation providers",
	Long: `Manage authentication credentials for all supported providers.

Each provider (Gemini API, Vertex AI, AWS Bedrock) has specific credential
requirements and pricing. Use the subcommands to:

  list  - View all providers and their authentication status
  test  - Test if authentication works for a provider
  setup - Interactively configure credentials for a provider

Different providers offer different models with different pricing:
  - gemini/flash-2.5: $0.039/image via Gemini API
  - vertex/flash-3.1: Gemini 3.1 Flash tiered pricing via Vertex AI
  - bedrock/nova-canvas: $0.08/image via AWS Bedrock`,
	Example: `  # List all providers and auth status
  gimage auth list

  # Set up Gemini authentication
  gimage auth setup gemini

  # Test if Vertex AI works
  gimage auth test vertex/flash-3.1`,
}

func init() {
	rootCmd.AddCommand(authCmd)
	// Provider-based commands
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authTestCmd)
	authCmd.AddCommand(authSetupCmd)
}
