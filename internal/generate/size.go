package generate

import (
	"math"
	"strconv"
	"strings"
)

// Common provider constraints
var (
	// Gemini 3 Pro / Gemini 3.1 Flash: Flexible but prefers certain ratios
	GoogleAspectRatios = []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "5:4", "4:5"}

	// GrokAspectRatios is the full set accepted by xAI Grok Imagine (generation + edits).
	// See https://docs.x.ai/developers/model-capabilities/images/generation
	GrokAspectRatios = []string{
		"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3",
		"2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20", "auto",
	}
)

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
