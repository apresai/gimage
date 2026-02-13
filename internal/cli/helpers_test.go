package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatImageSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 500, "500 B"},
		{"just under 1KB", 1023, "1023 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"1.5KB", 1536, "1.5 KB"},
		{"exactly 1MB", 1048576, "1.0 MB"},
		{"exactly 1GB", 1073741824, "1.0 GB"},
		{"2.5MB", 2621440, "2.5 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatImageSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"pad shorter string", "hello", 10, "hello     "},
		{"exact width", "hello", 5, "hello"},
		{"longer than width", "hello", 3, "hello"},
		{"empty string", "", 5, "     "},
		{"width zero", "hello", 0, "hello"},
		{"single char", "x", 4, "x   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.input, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadRight_WithANSI(t *testing.T) {
	// ANSI codes should not count toward visible width
	// "\033[32mhi\033[0m" has visible length 2
	ansiStr := "\033[32mhi\033[0m"
	result := padRight(ansiStr, 10)

	// Visible width is 2, so we need 8 spaces of padding
	expected := "\033[32mhi\033[0m        "
	assert.Equal(t, expected, result)
}

func TestPadRight_WithANSI_ExactWidth(t *testing.T) {
	ansiStr := "\033[31mOK\033[0m"
	result := padRight(ansiStr, 2)
	// Visible width equals requested width, no padding needed
	assert.Equal(t, ansiStr, result)
}
