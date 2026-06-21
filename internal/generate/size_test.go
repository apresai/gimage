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

func TestNormalizeDimensions(t *testing.T) {
	// multipleOf64Constraints exercises clamping and multiple-of rounding.
	multipleOf64Constraints := APISizeConstraints{
		MinWidth:   512,
		MaxWidth:   2048,
		MinHeight:  512,
		MaxHeight:  2048,
		MultipleOf: 64,
	}

	tests := []struct {
		name        string
		width       int
		height      int
		constraints APISizeConstraints
		wantWidth   int
		wantHeight  int
	}{
		{
			name:        "already valid",
			width:       1024,
			height:      1024,
			constraints: multipleOf64Constraints,
			wantWidth:   1024,
			wantHeight:  1024,
		},
		{
			name:        "below minimum clamped",
			width:       100,
			height:      100,
			constraints: multipleOf64Constraints,
			wantWidth:   512,
			wantHeight:  512,
		},
		{
			name:        "above maximum clamped",
			width:       3000,
			height:      3000,
			constraints: multipleOf64Constraints,
			wantWidth:   2048,
			wantHeight:  2048,
		},
		{
			name:        "rounded to nearest multiple of 64",
			width:       1000,
			height:      1000,
			constraints: multipleOf64Constraints,
			wantWidth:   1024,
			wantHeight:  1024,
		},
		{
			name:        "550 rounds to 576",
			width:       550,
			height:      550,
			constraints: multipleOf64Constraints,
			wantWidth:   576,
			wantHeight:  576,
		},
		{
			name:        "no constraints passthrough",
			width:       999,
			height:      777,
			constraints: APISizeConstraints{},
			wantWidth:   999,
			wantHeight:  777,
		},
		{
			name:   "round up near max stays in bounds",
			width:  2040,
			height: 2040,
			constraints: APISizeConstraints{
				MinWidth:   512,
				MaxWidth:   2048,
				MinHeight:  512,
				MaxHeight:  2048,
				MultipleOf: 64,
			},
			wantWidth:  2048,
			wantHeight: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := NormalizeDimensions(tt.width, tt.height, tt.constraints)
			assert.Equal(t, tt.wantWidth, w, "width")
			assert.Equal(t, tt.wantHeight, h, "height")
		})
	}
}

func TestRoundToMultiple(t *testing.T) {
	tests := []struct {
		name string
		n    int
		m    int
		want int
	}{
		{"rounds up when remainder > half", 100, 64, 128},
		{"already a multiple", 512, 64, 512},
		{"rounds down when remainder < half", 520, 64, 512},
		{"zero value", 0, 64, 0},
		{"m is zero returns n", 100, 0, 100},
		{"exact half rounds up", 96, 64, 128},
		{"just above half rounds up", 97, 64, 128},
		{"just below half rounds down", 95, 64, 64},
		{"multiple of 1", 100, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundToMultiple(tt.n, tt.m)
			assert.Equal(t, tt.want, got)
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

func TestFormatInternalSize(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		want   string
	}{
		{"standard", 1024, 768, "1024x768"},
		{"zeros", 0, 0, "0x0"},
		{"square", 512, 512, "512x512"},
		{"large", 4096, 4096, "4096x4096"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatInternalSize(tt.width, tt.height)
			assert.Equal(t, tt.want, got)
		})
	}
}
