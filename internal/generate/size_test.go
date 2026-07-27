package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantWidth  int
		wantHeight int
	}{
		{"standard square", "1024x1024", 1024, 1024},
		{"rectangular", "512x768", 512, 768},
		{"empty string defaults", "", 1024, 1024},
		{"invalid string defaults", "invalid", 1024, 1024},
		{"single number defaults", "1024", 1024, 1024},
		{"negative values default", "-1x-1", 1024, 1024},
		{"zero width defaults width", "0x768", 1024, 768},
		{"zero height defaults height", "512x0", 512, 1024},
		{"large values", "4096x4096", 4096, 4096},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := ParseSizeString(tt.input)
			assert.Equal(t, tt.wantWidth, w, "width")
			assert.Equal(t, tt.wantHeight, h, "height")
		})
	}
}

func TestInferAspectRatio(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		want   string
	}{
		{"square", 1024, 1024, "1:1"},
		{"16:9 landscape", 1920, 1080, "16:9"},
		{"9:16 portrait", 1080, 1920, "9:16"},
		{"zero dimensions default", 0, 0, "1:1"},
		{"4:3 landscape", 800, 600, "4:3"},
		{"3:4 portrait", 600, 800, "3:4"},
		{"negative width default", -1, 100, "1:1"},
		{"negative height default", 100, -1, "1:1"},
		{"3:2 landscape", 1500, 1000, "3:2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferAspectRatio(tt.width, tt.height, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInferAspectRatioCustomRatios(t *testing.T) {
	customRatios := []string{"1:1", "16:9"}

	tests := []struct {
		name   string
		width  int
		height int
		want   string
	}{
		{"exact 1:1", 1024, 1024, "1:1"},
		{"exact 16:9", 1920, 1080, "16:9"},
		{"closer to 16:9 than 1:1", 1600, 1000, "16:9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferAspectRatio(tt.width, tt.height, customRatios)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInferAspectRatioEmptyRatios(t *testing.T) {
	// Empty slice should use GoogleAspectRatios defaults
	got := InferAspectRatio(1920, 1080, []string{})
	assert.Equal(t, "16:9", got)
}
