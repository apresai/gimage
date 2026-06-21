package cli

import (
	"fmt"
	"os"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "0.1.0"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gimage",
	Short: "AI-powered image generation and processing CLI",
	Run: func(cmd *cobra.Command, args []string) {
		// Show help when run with no arguments
		cmd.Help()
	},
	Long: `gimage is a Go-based CLI tool for AI-powered image generation and processing.

FEATURES:
  • Generate images from text using Google Gemini 2.5 Flash Image or Gemini 3.1 Flash via Vertex
  • Process images: resize, scale, crop, compress (PNG, JPG, WebP, GIF, TIFF, BMP)
  • MCP server for Claude integration with batch operations
  • Pure Go implementation - single binary, no system dependencies

AUTHENTICATION:
  Before generating images, set up your API credentials:
    gimage auth gemini          # Setup Gemini API (simple, affordable)
    gimage auth vertex          # Setup Vertex AI (Express or Full mode)

  Get your Gemini API key from: https://aistudio.google.com/app/apikey

AVAILABLE COMMANDS:
  generate    Generate images from text prompts using AI
  resize      Resize images to specific dimensions
  scale       Scale images by a factor (e.g., 0.5 for half, 2.0 for double)
  crop        Crop images to specific regions
  compress    Compress images to reduce file size
  convert     Convert images between formats (PNG, JPG, WebP, GIF, TIFF, BMP)
  auth        Configure API credentials
  serve       Start MCP server for Claude integration (includes batch operations)
  tui         Launch interactive terminal UI

EXAMPLES:

  Image Generation:
  ────────────────
  1. Generate with default settings (Gemini):
     $ gimage generate "a sunset over mountains"

  2. Generate with specific model:
     $ gimage generate "futuristic city" --model gemini-3-pro-image

  3. Generate photorealistic image:
     $ gimage generate "portrait of a wise old wizard" --style photorealistic --size 1024x1024

  4. Generate with negative prompts:
     $ gimage generate "peaceful forest" --negative "people, buildings, roads"

  5. Generate reproducible image with seed:
     $ gimage generate "abstract geometric patterns" --seed 42

  6. Generate using Gemini 3.1 Flash via Vertex:
     $ gimage generate "hyper-realistic dragon" --model vertex-flash --project my-gcp-project

  7. List all available models:
     $ gimage generate --list-models

  8. Generate anime-style image:
     $ gimage generate "cherry blossoms in spring" --style anime

  Image Resizing:
  ──────────────
  9. Resize to specific dimensions:
     $ gimage resize photo.jpg 1920 1080

  10. Resize and save to custom output:
      $ gimage resize photo.png 800 600 --output thumbnail.png

  11. Resize maintaining aspect ratio (use scale instead):
      $ gimage scale photo.jpg 0.5

  Image Conversion:
  ────────────────
  12. Convert PNG to JPEG:
      $ gimage convert image.png jpg

  13. Convert to WebP for web optimization:
      $ gimage convert photo.jpg webp --output optimized.webp

  14. Convert to high-quality PNG:
      $ gimage convert photo.jpg png

  Image Compression:
  ─────────────────
  15. Compress image with quality setting:
      $ gimage compress large-photo.jpg --quality 85

  16. Compress PNG losslessly:
      $ gimage compress screenshot.png --output compressed.png

  Image Cropping:
  ──────────────
  17. Crop specific region (x=100, y=100, width=800, height=600):
      $ gimage crop photo.jpg 100 100 800 600

  18. Crop and save:
      $ gimage crop image.png 0 0 1920 1080 --output cropped.png

  MCP Server (for batch operations via Claude):
  ──────────────────────────────────────────────
  19. Start MCP server:
      $ gimage serve

  20. Batch operations available via MCP tools:
      - batch_resize, batch_compress, batch_convert
      See: https://github.com/apresai/gimage/blob/main/docs/MCP_TOOLS.md

  Advanced Usage:
  ──────────────
  21. Generate large 2K image with Gemini 3.1 Flash via Vertex:
      $ gimage generate "detailed landscape" --model vertex-flash --image-size 2K --project my-project

  22. Chain operations (generate, then resize):
      $ gimage generate "logo design" --output logo.png
      $ gimage resize logo.png 512 512 --output logo-512.png

  23. Verbose mode for debugging:
      $ gimage generate "test image" --verbose

  24. Launch interactive TUI:
      $ gimage tui

TIPS:
  • Use --verbose flag for detailed operation logs
  • Use --help on any command for more details (e.g., "gimage generate --help")
  • Default output filenames are auto-generated with timestamps
  • For batch operations, use MCP server with Claude Desktop
  • All image processing uses high-quality Lanczos resampling
  • Config file location: ~/.gimage/config.md

For more information, visit: https://github.com/apresai/gimage`,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Initialize logging
	logger := logging.GetLogger()
	if logger.IsEnabled() {
		logAuthStatus(logger)
	}

	err := rootCmd.Execute()
	if err != nil {
		logger.LogError("Command execution failed: %v", err)
		os.Exit(1)
	}

	// Close logger on exit
	logger.Close()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gimage/config.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gimage" (without extension).
		viper.AddConfigPath(home + "/.gimage")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && viper.GetBool("verbose") {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// logAuthStatus logs which LLM authentication methods are available
func logAuthStatus(logger *logging.Logger) {
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.LogWarn("Failed to load config for auth status check: %v", err)
		return
	}

	// Check Gemini auth
	if cfg.GeminiAPIKey != "" {
		logger.LogAuthStatus("Gemini", true, "API key configured")
	} else {
		logger.LogAuthStatus("Gemini", false, "No API key configured")
	}

	// Check Vertex auth
	if cfg.VertexAPIKey != "" || cfg.VertexProject != "" {
		logger.LogAuthStatus("Vertex AI", true, fmt.Sprintf("Project: %s, Location: %s", cfg.VertexProject, cfg.VertexLocation))
	} else {
		logger.LogAuthStatus("Vertex AI", false, "No credentials configured")
	}

	// Check Bedrock auth
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		logger.LogAuthStatus("AWS Bedrock", true, fmt.Sprintf("Region: %s", cfg.AWSRegion))
	} else {
		logger.LogAuthStatus("AWS Bedrock", false, "No credentials configured")
	}

	// Log available providers
	logger.LogInfo("Available providers:")
	registry := generate.GetProviderRegistry()
	for _, p := range registry.List() {
		logger.LogInfo("  - %s (API: %s, Free: %v)", p.ID, p.API, p.Pricing.FreeTier)
	}
}
