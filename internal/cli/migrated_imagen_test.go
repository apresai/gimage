package cli

import (
	"testing"

	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
)

func TestIsMigratedImagenProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider *generate.Provider
		want     bool
	}{
		{"nil", nil, false},
		{"imagen-4", &generate.Provider{ID: "vertex/imagen-4"}, true},
		{"imagen-4-fast", &generate.Provider{ID: "vertex/imagen-4-fast"}, true},
		{"imagen-4-ultra", &generate.Provider{ID: "vertex/imagen-4-ultra"}, true},
		{"gemini", &generate.Provider{ID: "gemini/flash-2.5"}, false},
		{"grok", &generate.Provider{ID: "grok/imagine"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMigratedImagenProvider(tc.provider); got != tc.want {
				t.Fatalf("isMigratedImagenProvider(%v) = %v, want %v", tc.provider, got, tc.want)
			}
		})
	}
}

func TestDefaultThinkingLevelForProvider(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"vertex/imagen-4-fast", "minimal"},
		{"vertex/imagen-4", "medium"},
		{"vertex/imagen-4-ultra", "high"},
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

func TestStripUnsupportedMigratedImagenOptions(t *testing.T) {
	// Migrated provider: negative prompt + seed are cleared.
	opts := &models.GenerateOptions{NegativePrompt: "people", Seed: 42}
	stripUnsupportedMigratedImagenOptions(&generate.Provider{ID: "vertex/imagen-4-ultra"}, opts)
	if opts.NegativePrompt != "" || opts.Seed != 0 {
		t.Fatalf("expected negative/seed cleared, got %q / %d", opts.NegativePrompt, opts.Seed)
	}

	// Non-migrated provider: options are preserved.
	keep := &models.GenerateOptions{NegativePrompt: "people", Seed: 42}
	stripUnsupportedMigratedImagenOptions(&generate.Provider{ID: "gemini/flash-2.5"}, keep)
	if keep.NegativePrompt != "people" || keep.Seed != 42 {
		t.Fatalf("expected options preserved for non-migrated provider, got %q / %d", keep.NegativePrompt, keep.Seed)
	}

	// Nil guards must not panic.
	stripUnsupportedMigratedImagenOptions(nil, keep)
	stripUnsupportedMigratedImagenOptions(&generate.Provider{ID: "vertex/imagen-4"}, nil)
}

func TestWarnIgnoredMigratedImagenOptions_NoPanic(t *testing.T) {
	// Smoke test: exercises every warning branch; must not panic.
	warnIgnoredMigratedImagenOptions(&generate.Provider{ID: "vertex/imagen-4"}, "people", 7)
	warnIgnoredMigratedImagenOptions(&generate.Provider{ID: "vertex/imagen-4"}, "", 0)
	warnIgnoredMigratedImagenOptions(&generate.Provider{ID: "gemini/flash-2.5"}, "people", 7)
	warnIgnoredMigratedImagenOptions(nil, "people", 7)
}
