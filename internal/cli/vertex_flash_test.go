package cli

import (
	"testing"

	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
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
		{"grok", &generate.Provider{ID: "grok/imagine"}, false},
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
		{"grok/imagine", ""},
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

func TestStripUnsupportedVertexFlashOptions(t *testing.T) {
	// Vertex Flash provider: negative prompt + seed are cleared.
	opts := &models.GenerateOptions{NegativePrompt: "people", Seed: 42}
	stripUnsupportedVertexFlashOptions(&generate.Provider{ID: "vertex/flash-3.1-ultra"}, opts)
	if opts.NegativePrompt != "" || opts.Seed != 0 {
		t.Fatalf("expected negative/seed cleared, got %q / %d", opts.NegativePrompt, opts.Seed)
	}

	// Non-vertex-flash provider: options are preserved.
	keep := &models.GenerateOptions{NegativePrompt: "people", Seed: 42}
	stripUnsupportedVertexFlashOptions(&generate.Provider{ID: "gemini/flash-2.5"}, keep)
	if keep.NegativePrompt != "people" || keep.Seed != 42 {
		t.Fatalf("expected options preserved for non-vertex-flash provider, got %q / %d", keep.NegativePrompt, keep.Seed)
	}

	// Nil guards must not panic.
	stripUnsupportedVertexFlashOptions(nil, keep)
	stripUnsupportedVertexFlashOptions(&generate.Provider{ID: "vertex/flash-3.1"}, nil)
}

func TestWarnIgnoredVertexFlashOptions_NoPanic(t *testing.T) {
	// Smoke test: exercises every warning branch; must not panic.
	warnIgnoredVertexFlashOptions(&generate.Provider{ID: "vertex/flash-3.1"}, "people", 7)
	warnIgnoredVertexFlashOptions(&generate.Provider{ID: "vertex/flash-3.1"}, "", 0)
	warnIgnoredVertexFlashOptions(&generate.Provider{ID: "gemini/flash-2.5"}, "people", 7)
	warnIgnoredVertexFlashOptions(nil, "people", 7)
}
