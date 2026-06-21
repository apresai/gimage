package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
	"github.com/spf13/cobra"
)

// authTestCmd tests authentication for a specific provider
var authTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Test authentication for a specific provider",
	Long: `Tests if authentication works for a specific provider by attempting
to connect to the API and optionally generating a test image.

Provider can be specified as:
- Full ID: gemini/flash-2.5, vertex/flash-3.1
- Alias: gemini, vertex-flash, nova
- Pattern: gemini/* (tests all Gemini providers)`,
	Example: `  # Test Gemini authentication
  gimage auth test gemini/flash-2.5

  # Test using alias
  gimage auth test gemini

  # Test with actual generation
  gimage auth test gemini --generate

  # Test all providers
  gimage auth test --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAuthTest,
}

var (
	testAll      bool
	testGenerate bool
	testVerbose  bool
)

func init() {
	authTestCmd.Flags().BoolVar(&testAll, "all", false, "Test all providers")
	authTestCmd.Flags().BoolVar(&testGenerate, "generate", false, "Actually generate a test image")
	authTestCmd.Flags().BoolVar(&testVerbose, "verbose", false, "Show detailed test output")
}

func runAuthTest(cmd *cobra.Command, args []string) error {
	registry := generate.GetProviderRegistry()

	var providers []*generate.Provider

	if testAll {
		providers = registry.List()
	} else {
		if len(args) == 0 {
			return fmt.Errorf("provider ID required (or use --all)")
		}

		// Resolve provider
		provider, err := registry.ResolveProvider(args[0])
		if err != nil {
			// Check if it's a pattern like "gemini/*"
			if args[0] == "gemini/*" || args[0] == "gemini" {
				providers = registry.ListByAPI("gemini")
			} else if args[0] == "vertex/*" || args[0] == "vertex" {
				providers = registry.ListByAPI("vertex")
			} else if args[0] == "bedrock/*" || args[0] == "bedrock" {
				providers = registry.ListByAPI("bedrock")
			} else {
				return err
			}
		} else {
			providers = []*generate.Provider{provider}
		}
	}

	if len(providers) == 0 {
		return fmt.Errorf("no providers found")
	}

	// Test each provider
	allPassed := true
	for i, provider := range providers {
		if i > 0 {
			fmt.Println()
		}

		passed := testProvider(registry, provider)
		if !passed {
			allPassed = false
		}
	}

	if !allPassed {
		return fmt.Errorf("some tests failed")
	}

	return nil
}

func testProvider(registry *generate.ProviderRegistry, provider *generate.Provider) bool {
	fmt.Printf("Testing: %s\n", provider.Name)
	fmt.Printf("Provider ID: %s\n", provider.ID)

	// Check credentials
	hasAuth, missing, err := registry.CheckAuth(provider)
	if err != nil {
		fmt.Printf("  ✗ Error checking auth: %v\n", err)
		return false
	}

	if !hasAuth {
		fmt.Printf("  ✗ Missing credentials: %v\n", missing)
		fmt.Println("  → Use 'gimage auth setup " + provider.ID + "' to configure")
		return false
	}

	fmt.Println("  ✓ All required credentials found")

	// Try to create client
	fmt.Print("  • Creating client... ")
	start := time.Now()
	client, err := registry.CreateClient(provider.ID)
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return false
	}
	defer client.Close()
	fmt.Printf("✓ Success (%.2fs)\n", time.Since(start).Seconds())

	// Test generation if requested
	if testGenerate {
		fmt.Print("  • Generating test image... ")
		start = time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		options := models.GenerateOptions{
			Model: provider.ModelID,
			Size:  "256x256", // Small size for testing
		}

		_, err = client.GenerateImage(ctx, "simple test image", options)
		if err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
			if testVerbose {
				fmt.Printf("    Full error: %+v\n", err)
			}
			return false
		}
		fmt.Printf("✓ Success (%.2fs)\n", time.Since(start).Seconds())
	}

	// Show pricing info
	if provider.Pricing.FreeTier {
		fmt.Printf("  • Pricing: FREE (%s)\n", provider.Pricing.FreeTierLimit)
	} else if provider.Pricing.CostPerImage != nil {
		fmt.Printf("  • Pricing: $%.4f per image\n", *provider.Pricing.CostPerImage)
	}

	fmt.Printf("  ✓ Provider %s is ready to use!\n", provider.ID)
	return true
}
