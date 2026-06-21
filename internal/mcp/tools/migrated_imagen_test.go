package tools

import (
	"strings"
	"testing"

	"github.com/apresai/gimage/internal/generate"
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

func TestMigratedImagenWarning(t *testing.T) {
	p := &generate.Provider{ID: "vertex/imagen-4"}

	// Both negative and seed set -> warning mentions both.
	w := migratedImagenWarning(p, "people", 5)
	if !strings.Contains(w, "negative") || !strings.Contains(w, "seed") {
		t.Fatalf("expected warning mentioning negative and seed, got %q", w)
	}

	// Only negative set.
	if w := migratedImagenWarning(p, "people", 0); !strings.Contains(w, "negative") || strings.Contains(w, "seed") {
		t.Fatalf("expected negative-only warning, got %q", w)
	}

	// Neither set -> empty.
	if w := migratedImagenWarning(p, "", 0); w != "" {
		t.Fatalf("expected empty warning, got %q", w)
	}

	// Non-migrated provider -> empty regardless of options.
	if w := migratedImagenWarning(&generate.Provider{ID: "gemini/flash-2.5"}, "people", 5); w != "" {
		t.Fatalf("expected empty for non-migrated provider, got %q", w)
	}
}
