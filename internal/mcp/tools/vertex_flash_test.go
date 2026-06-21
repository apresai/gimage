package tools

import (
	"strings"
	"testing"

	"github.com/apresai/gimage/internal/generate"
)

func TestIsVertexFlashProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider *generate.Provider
		want     bool
	}{
		{"nil", nil, false},
		{"vertex-flash", &generate.Provider{ID: "vertex/flash-3.1"}, true},
		{"vertex-flash-fast", &generate.Provider{ID: "vertex/flash-3.1-fast"}, true},
		{"vertex-flash-ultra", &generate.Provider{ID: "vertex/flash-3.1-ultra"}, true},
		{"gemini", &generate.Provider{ID: "gemini/flash-2.5"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isVertexFlashProvider(tc.provider); got != tc.want {
				t.Fatalf("isVertexFlashProvider(%v) = %v, want %v", tc.provider, got, tc.want)
			}
		})
	}
}

func TestDefaultThinkingLevelForProvider(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"vertex/flash-3.1-fast", "minimal"},
		{"vertex/flash-3.1", "medium"},
		{"vertex/flash-3.1-ultra", "high"},
		{"gemini/flash-2.5", ""},
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

func TestVertexFlashWarning(t *testing.T) {
	p := &generate.Provider{ID: "vertex/flash-3.1"}

	// Both negative and seed set -> warning mentions both.
	w := vertexFlashWarning(p, "people", 5)
	if !strings.Contains(w, "negative") || !strings.Contains(w, "seed") {
		t.Fatalf("expected warning mentioning negative and seed, got %q", w)
	}

	// Only negative set.
	if w := vertexFlashWarning(p, "people", 0); !strings.Contains(w, "negative") || strings.Contains(w, "seed") {
		t.Fatalf("expected negative-only warning, got %q", w)
	}

	// Neither set -> empty.
	if w := vertexFlashWarning(p, "", 0); w != "" {
		t.Fatalf("expected empty warning, got %q", w)
	}

	// Non-vertex-flash provider -> empty regardless of options.
	if w := vertexFlashWarning(&generate.Provider{ID: "gemini/flash-2.5"}, "people", 5); w != "" {
		t.Fatalf("expected empty for non-vertex-flash provider, got %q", w)
	}
}
