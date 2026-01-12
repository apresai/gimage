package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apresai/gimage/internal/imaging"
	"github.com/spf13/cobra"
)

// resizeCmd represents the resize command
var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Resize an image to specific dimensions",
	Long: `Resize an image to specific dimensions using high-quality Lanczos resampling.

Examples:
  gimage resize --input input.jpg --width 800 --height 600 --mode crop
  gimage resize -i input.png -w 1920 -h 1080 --mode fit -o resized.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flag values
		inputPath, _ := cmd.Flags().GetString("input")
		width, _ := cmd.Flags().GetInt("width")
		height, _ := cmd.Flags().GetInt("height")
		outputPath, _ := cmd.Flags().GetString("output")
		mode, _ := cmd.Flags().GetString("mode")

		// Validate required flags
		if inputPath == "" {
			return fmt.Errorf("--input flag is required")
		}
		if width <= 0 {
			return fmt.Errorf("--width must be a positive integer")
		}
		if height <= 0 {
			return fmt.Errorf("--height must be a positive integer")
		}

		// Get output path or generate default
		if outputPath == "" {
			// Generate output path: input_resized_WxH.ext
			ext := filepath.Ext(inputPath)
			base := inputPath[:len(inputPath)-len(ext)]
			outputPath = fmt.Sprintf("%s_resized_%dx%d%s", base, width, height, ext)
		}

		// Validate input file exists
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", inputPath)
		}

		ctx := context.Background()
		var err error
		if mode == "fit" {
			printInfo("Resizing %s to fit within %dx%d (preserving aspect ratio)...", inputPath, width, height)
			err = imaging.ResizeFit(ctx, inputPath, outputPath, width, height)
		} else {
			printInfo("Resizing %s to %dx%d (crop/fill, preserving aspect ratio)...", inputPath, width, height)
			err = imaging.ResizeFill(ctx, inputPath, outputPath, width, height)
		}

		if err != nil {
			return fmt.Errorf("resize failed: %w", err)
		}

		// Get file size for reporting
		info, err := os.Stat(outputPath)
		if err != nil {
			return fmt.Errorf("failed to stat output file: %w", err)
		}

		printSuccess("Resized successfully!")
		printInfo("Output: %s", outputPath)
		printInfo("Size: %d bytes", info.Size())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resizeCmd)

	// Flags for resize command
	resizeCmd.Flags().StringP("input", "i", "", "input image file path (required)")
	resizeCmd.Flags().Int("width", 0, "target width in pixels (required)")
	resizeCmd.Flags().Int("height", 0, "target height in pixels (required)")
	resizeCmd.Flags().StringP("output", "o", "", "output file path (default: input_resized_WxH.ext)")
	resizeCmd.Flags().String("mode", "crop", "Resize mode: crop (fills size), fit (fits inside) (default: crop)")

	resizeCmd.MarkFlagRequired("input")
	resizeCmd.MarkFlagRequired("width")
	resizeCmd.MarkFlagRequired("height")
}
