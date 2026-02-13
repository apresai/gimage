package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/mcp/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for AI assistant integration",
	Long: `Start the gimage MCP (Model Context Protocol) server.

This allows AI assistants like Claude to use gimage for image generation
and processing operations. The server communicates over stdio using the
MCP protocol.

USAGE WITH CLAUDE DESKTOP:

Add this to your Claude Desktop MCP configuration:

macOS/Linux:
  {
    "mcpServers": {
      "gimage": {
        "command": "gimage",
        "args": ["serve"]
      }
    }
  }

Configuration file location:
  • macOS: ~/Library/Application Support/Claude/claude_desktop_config.json
  • Linux: ~/.config/Claude/claude_desktop_config.json
  • Windows: %APPDATA%\Claude\claude_desktop_config.json

ENVIRONMENT VARIABLES:

The serve command respects the same environment variables as the CLI:
  • GEMINI_API_KEY - Gemini API key for image generation
  • VERTEX_API_KEY - Vertex AI API key (Express Mode)
  • VERTEX_PROJECT - GCP project ID for Vertex AI
  • VERTEX_LOCATION - Vertex AI location (default: us-central1)
  • GOOGLE_APPLICATION_CREDENTIALS - Path to service account JSON

FEATURES:

The MCP server exposes 10 tools to AI assistants:
  • generate_image    - AI image generation with Gemini/Vertex
  • resize_image      - Resize to specific dimensions
  • scale_image       - Scale by factor
  • crop_image        - Crop to region
  • compress_image    - Compress to reduce file size
  • convert_image     - Convert between formats
  • batch_resize      - Batch resize operations
  • batch_compress    - Batch compression
  • batch_convert     - Batch format conversion
  • list_models       - List available AI models

EXAMPLES:

  # Start MCP server (usually called by AI assistant)
  $ gimage serve

  # Test with verbose logging (logs go to stderr)
  $ gimage serve --verbose

  # Use custom config file
  $ gimage serve --config ~/.gimage/custom-config.yaml

TROUBLESHOOTING:

If the MCP server isn't working in Claude:
  1. Check that gimage is in your PATH: which gimage
  2. Verify your API keys are configured: gimage auth gemini
  3. Look at Claude's logs for error messages
  4. Test image generation works: gimage generate "test image"

For more information: https://github.com/apresai/gimage`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration (optional - tools will load it when needed)
		cfg, err := config.LoadConfig()
		if err != nil {
			// Non-fatal error - server can still work without config for some operations
			fmt.Fprintf(os.Stderr, "[gimage-mcp] Warning: failed to load config: %v\n", err)
			cfg = &config.Config{} // Use empty config
		}

		// Get verbose flag
		verbose := viper.GetBool("verbose")

		// Create MCP server
		server := mcp.NewMCPServer("gimage", version, cfg, verbose)

		// Register all tools
		tools.RegisterGenerateImageTool(server)
		tools.RegisterResizeImageTool(server)
		tools.RegisterScaleImageTool(server)
		tools.RegisterCropImageTool(server)
		tools.RegisterCompressImageTool(server)
		tools.RegisterConvertImageTool(server)
		tools.RegisterBatchResizeTool(server)
		tools.RegisterBatchCompressTool(server)
		tools.RegisterBatchConvertTool(server)
		tools.RegisterListModelsTool(server)

		// Register all prompts (examples and learning templates for LLMs)
		mcp.RegisterAllPrompts(server)

		// Setup signal handling for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Fprintln(os.Stderr, "\n[gimage-mcp] Shutting down gracefully...")
			cancel()
		}()

		// Log startup (to stderr, not stdout - stdout is for JSON-RPC)
		if verbose {
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Starting MCP server")
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Protocol: Model Context Protocol")
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Transport: stdio")
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Tools: 10 registered")
			fmt.Fprintln(os.Stderr, "")

			// Show available providers with pricing
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Discovering available providers...")

			registry := generate.GetProviderRegistry()
			statuses := registry.GetAuthStatus()

			// Count configured providers
			configuredCount := 0
			for _, status := range statuses {
				if status.Configured {
					configuredCount++
				}
			}

			if configuredCount == 0 {
				fmt.Fprintln(os.Stderr, "[gimage-mcp] ⚠️  No providers configured")
				fmt.Fprintln(os.Stderr, "[gimage-mcp]     Run: gimage auth setup gemini/flash-2.5  ($0.039/image, affordable)")
				fmt.Fprintln(os.Stderr, "[gimage-mcp]     Or:  gimage auth setup vertex/imagen-4  (Paid, highest quality)")
			} else {
				fmt.Fprintf(os.Stderr, "[gimage-mcp] 📊 Available Providers (%d configured):\n", configuredCount)
				fmt.Fprintln(os.Stderr, "")

				// Group by API
				geminiProviders := []generate.AuthStatus{}
				vertexProviders := []generate.AuthStatus{}
				bedrockProviders := []generate.AuthStatus{}

				for _, status := range statuses {
					if !status.Configured {
						continue
					}
					switch status.Provider.API {
					case "gemini":
						geminiProviders = append(geminiProviders, status)
					case "vertex":
						vertexProviders = append(vertexProviders, status)
					case "bedrock":
						bedrockProviders = append(bedrockProviders, status)
					}
				}

				if len(geminiProviders) > 0 {
					fmt.Fprintf(os.Stderr, "[gimage-mcp] ✓ Gemini API - %d provider(s) configured\n", len(geminiProviders))
					for _, status := range geminiProviders {
						p := status.Provider
						pricingInfo := "Variable"
						if p.Pricing.FreeTier {
							pricingInfo = fmt.Sprintf("FREE (%s)", p.Pricing.FreeTierLimit)
						} else if p.Pricing.CostPerImage != nil {
							pricingInfo = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
						}
						fmt.Fprintf(os.Stderr, "[gimage-mcp]   • %s - %s\n", p.Name, pricingInfo)
					}
					fmt.Fprintln(os.Stderr, "")
				}

				if len(vertexProviders) > 0 {
					fmt.Fprintf(os.Stderr, "[gimage-mcp] ✓ Vertex AI - %d provider(s) configured\n", len(vertexProviders))
					for _, status := range vertexProviders {
						p := status.Provider
						pricingInfo := "Variable"
						if p.Pricing.CostPerImage != nil {
							pricingInfo = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
						}
						fmt.Fprintf(os.Stderr, "[gimage-mcp]   • %s - %s\n", p.Name, pricingInfo)
					}
					fmt.Fprintln(os.Stderr, "")
				}

				if len(bedrockProviders) > 0 {
					fmt.Fprintf(os.Stderr, "[gimage-mcp] ✓ AWS Bedrock - %d provider(s) configured\n", len(bedrockProviders))
					for _, status := range bedrockProviders {
						p := status.Provider
						pricingInfo := "Variable"
						if p.Pricing.CostPerImage != nil {
							pricingInfo = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
						}
						fmt.Fprintf(os.Stderr, "[gimage-mcp]   • %s - %s\n", p.Name, pricingInfo)
					}
					fmt.Fprintln(os.Stderr, "")
				}
			}

			// Show default provider (first configured, preferring free tier)
			var defaultProvider *generate.Provider
			for _, status := range statuses {
				if status.Configured {
					defaultProvider = status.Provider
					if status.Provider.Pricing.FreeTier {
						break // Prefer free tier
					}
				}
			}

			if defaultProvider != nil {
				pricingInfo := "Variable"
				if defaultProvider.Pricing.FreeTier {
					pricingInfo = fmt.Sprintf("FREE (%s)", defaultProvider.Pricing.FreeTierLimit)
				} else if defaultProvider.Pricing.CostPerImage != nil {
					pricingInfo = fmt.Sprintf("$%.4f/image", *defaultProvider.Pricing.CostPerImage)
				}
				fmt.Fprintf(os.Stderr, "[gimage-mcp] 🎯 Default Provider: %s\n", defaultProvider.Name)
				fmt.Fprintf(os.Stderr, "[gimage-mcp]    %s\n", pricingInfo)
			} else {
				fmt.Fprintln(os.Stderr, "[gimage-mcp] ⚠️  No default provider available - missing credentials")
			}

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "[gimage-mcp] 🎧 Ready for requests...")
		}

		// Start server
		if err := server.Start(ctx); err != nil && err != context.Canceled {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
