package tools

import (
	"testing"

	"github.com/apresai/gimage/internal/generate"
)

func TestDefaultThinkingLevelForProvider(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"vertex/flash-3.1-fast", "minimal"},
		{"vertex/flash-3.1-lite", "minimal"},
		{"vertex/flash-3.1", "medium"},
		{"vertex/flash-3.1-ultra", "high"},
		{"gemini/flash-2.5", ""},
		{"gemini/flash-3.1-lite", ""},
		{"grok/grok-imagine", ""},
		{"grok/grok-imagine-2.0", ""},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			if got := defaultThinkingLevelForProvider(&generate.Provider{ID: tc.id}); got != tc.want {
				t.Fatalf("defaultThinkingLevelForProvider(%q) = %q, want %q", tc.id, got, tc.want)
			}
		})
	}
	if got := defaultThinkingLevelForProvider(nil); got != "" {
		t.Fatalf("defaultThinkingLevelForProvider(nil) = %q, want empty", got)
	}
}
