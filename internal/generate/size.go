package generate

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// APISizeConstraints defines dimension constraints for a specific AI provider
type APISizeConstraints struct {
	MinWidth    int
	MaxWidth    int
	MinHeight   int
	MaxHeight   int
	MultipleOf  int
	FixedRatios []string // e.g., ["1:1", "16:9", "4:3"]
}

// Common provider constraints
var (
	// Bedrock Nova Canvas: 512-2048, multiples of 64
	BedrockConstraints = APISizeConstraints{
		MinWidth:   512,
		MaxWidth:   2048,
		MinHeight:  512,
		MaxHeight:  2048,
		MultipleOf: 64,
	}

	// Gemini 3 Pro / Gemini 3.1 Flash: Flexible but prefers certain ratios
	GoogleAspectRatios = []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "5:4", "4:5"}
)

// NormalizeDimensions adjusts dimensions to meet API constraints using a "closest match" strategy
func NormalizeDimensions(width, height int, constraints APISizeConstraints) (int, int) {
	// 1. Clamp to Min/Max
	if constraints.MinWidth > 0 && width < constraints.MinWidth {
		width = constraints.MinWidth
	}
	if constraints.MaxWidth > 0 && width > constraints.MaxWidth {
		width = constraints.MaxWidth
	}
	if constraints.MinHeight > 0 && height < constraints.MinHeight {
		height = constraints.MinHeight
	}
	if constraints.MaxHeight > 0 && height > constraints.MaxHeight {
		height = constraints.MaxHeight
	}

	// 2. Round to closest MultipleOf
	if constraints.MultipleOf > 0 {
		width = roundToMultiple(width, constraints.MultipleOf)
		height = roundToMultiple(height, constraints.MultipleOf)

		// Re-clamp in case rounding pushed it out of bounds
		if constraints.MaxWidth > 0 && width > constraints.MaxWidth {
			width = width - constraints.MultipleOf
		}
		if constraints.MaxHeight > 0 && height > constraints.MaxHeight {
			height = height - constraints.MultipleOf
		}
		if constraints.MinWidth > 0 && width < constraints.MinWidth {
			width = width + constraints.MultipleOf
		}
		if constraints.MinHeight > 0 && height < constraints.MinHeight {
			height = height + constraints.MultipleOf
		}
	}

	return width, height
}

// roundToMultiple rounds n to the nearest multiple of m
func roundToMultiple(n, m int) int {
	if m <= 0 {
		return n
	}
	remainder := n % m
	if remainder == 0 {
		return n
	}
	if remainder >= m/2 {
		return n + (m - remainder)
	}
	return n - remainder
}

// ParseSizeString converts "WIDTHxHEIGHT" to (int, int) with defaults
func ParseSizeString(size string) (int, int) {
	if size == "" {
		return 1024, 1024
	}

	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 1024, 1024
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil || width <= 0 {
		width = 1024
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil || height <= 0 {
		height = 1024
	}

	return width, height
}

// InferAspectRatio determines the best matching aspect ratio for given dimensions
func InferAspectRatio(width, height int, supportedRatios []string) string {
	if width <= 0 || height <= 0 {
		return "1:1"
	}

	if len(supportedRatios) == 0 {
		supportedRatios = GoogleAspectRatios
	}

	ratio := float64(width) / float64(height)

	type aspectMatch struct {
		name  string
		value float64
	}

	var matches []aspectMatch
	for _, r := range supportedRatios {
		parts := strings.Split(r, ":")
		if len(parts) != 2 {
			continue
		}
		w, _ := strconv.ParseFloat(parts[0], 64)
		h, _ := strconv.ParseFloat(parts[1], 64)
		if h != 0 {
			matches = append(matches, aspectMatch{name: r, value: w / h})
		}
	}

	if len(matches) == 0 {
		return "1:1"
	}

	// Find closest match
	closest := matches[0]
	smallestDiff := math.Abs(ratio - closest.value)

	for _, m := range matches[1:] {
		diff := math.Abs(ratio - m.value)
		if diff < smallestDiff {
			smallestDiff = diff
			closest = m
		}
	}

	return closest.name
}

// FormatInternalSize converts dimensions back to "WIDTHxHEIGHT" string
func FormatInternalSize(width, height int) string {
	return fmt.Sprintf("%dx%d", width, height)
}
