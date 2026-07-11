package generate

import (
	"strings"
	"testing"

	"github.com/apresai/gimage/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullOptions returns a GenerateOptions with every reconcilable field set, so a
// test can assert precisely which fields a provider strips vs preserves.
func fullOptions() models.GenerateOptions {
	return models.GenerateOptions{
		Model:              "x",
		Size:               "1024x1024",
		Style:              "anime",
		Seed:               42,
		ImageSize:          "2K",
		AspectRatio:        "16:9",
		NumberOfImages:     1,
		OutputFormat:       "webp",
		ThinkingLevel:      "high",
		WebSearchGrounding: true,
		InputImages:        []string{"a.png"},
	}
}

func warningsContain(warnings []string, substr string) bool {
	for _, w := range warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}

func TestReconcileOptions_Grok(t *testing.T) {
	p, err := GetProviderRegistry().Get("grok/grok-imagine")
	require.NoError(t, err)

	o := fullOptions()
	o.NumberOfImages = 3 // Grok honors N exactly — must NOT be stripped or warned.
	w := ReconcileOptions(p, &o)

	// Grok ignores these — warned and stripped.
	assert.Empty(t, o.Style)
	assert.Zero(t, o.Seed)
	assert.Empty(t, o.ThinkingLevel)
	assert.False(t, o.WebSearchGrounding)
	assert.Empty(t, o.OutputFormat)
	assert.True(t, warningsContain(w, "--style"))
	assert.True(t, warningsContain(w, "--seed"))
	assert.True(t, warningsContain(w, "--thinking"))
	assert.True(t, warningsContain(w, "--grounding"))
	assert.True(t, warningsContain(w, "--output-format"))
	assert.False(t, warningsContain(w, "--input-image"))

	// Grok honors these — preserved, no warning.
	assert.Equal(t, "2K", o.ImageSize)
	assert.Equal(t, "16:9", o.AspectRatio)
	assert.Equal(t, 3, o.NumberOfImages)
	assert.Equal(t, []string{"a.png"}, o.InputImages)
	assert.False(t, warningsContain(w, "--image-size"))
	assert.False(t, warningsContain(w, "--aspect-ratio"))
	assert.False(t, warningsContain(w, "--count"))
}

func TestReconcileOptions_Gemini25(t *testing.T) {
	p, err := GetProviderRegistry().Get("gemini/flash-2.5")
	require.NoError(t, err)

	o := fullOptions()
	w := ReconcileOptions(p, &o)

	// 2.5 Flash does not do native image-size, thinking, or grounding.
	assert.Empty(t, o.ImageSize)
	assert.Empty(t, o.ThinkingLevel)
	assert.False(t, o.WebSearchGrounding)
	assert.True(t, warningsContain(w, "--image-size"))
	assert.True(t, warningsContain(w, "--thinking"))
	assert.True(t, warningsContain(w, "--grounding"))

	// 2.5 Flash honors these — preserved.
	assert.Equal(t, "anime", o.Style)
	assert.Equal(t, int64(42), o.Seed)
	assert.Equal(t, "16:9", o.AspectRatio)
	assert.Equal(t, []string{"a.png"}, o.InputImages)
}

func TestReconcileOptions_VertexFlash(t *testing.T) {
	p, err := GetProviderRegistry().Get("vertex/flash-3.1")
	require.NoError(t, err)

	o := fullOptions()
	w := ReconcileOptions(p, &o)

	// Vertex Flash does not support seed.
	assert.Zero(t, o.Seed)
	assert.True(t, warningsContain(w, "--seed"))

	// Vertex full-mode honors output format, thinking, image size — preserved.
	assert.Equal(t, "webp", o.OutputFormat)
	assert.Equal(t, "high", o.ThinkingLevel)
	assert.Equal(t, "2K", o.ImageSize)
	assert.False(t, warningsContain(w, "--output-format"))
}

func TestReconcileOptions_CountBestEffort(t *testing.T) {
	p, err := GetProviderRegistry().Get("gemini/pro-3")
	require.NoError(t, err)

	o := models.GenerateOptions{NumberOfImages: 2}
	w := ReconcileOptions(p, &o)

	// Count is never stripped, but the Gemini family gets a best-effort note.
	assert.Equal(t, 2, o.NumberOfImages)
	assert.True(t, warningsContain(w, "--count"))
}

func TestReconcileOptions_NilSafe(t *testing.T) {
	assert.Nil(t, ReconcileOptions(nil, &models.GenerateOptions{Style: "anime"}))
	assert.Nil(t, ReconcileOptions(&Provider{ID: "x"}, nil))
}
