package generate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apresai/gimage/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGrokImageRequest(t *testing.T) {
	const samplePrompt = "an astronaut riding a horse"

	tests := []struct {
		name           string
		opts           models.GenerateOptions
		wantModel      string
		wantResolution string
		wantAspect     string
	}{
		{
			name:           "empty model uses default",
			opts:           models.GenerateOptions{},
			wantModel:      "grok-imagine-image",
			wantResolution: "",
		},
		{
			name:           "1K image-size mapped to 1k",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", ImageSize: "1K"},
			wantModel:      "grok-imagine-image",
			wantResolution: "1k",
		},
		{
			name:           "2K image-size mapped to 2k on quality model",
			opts:           models.GenerateOptions{Model: "grok-imagine-image-quality", ImageSize: "2K"},
			wantModel:      "grok-imagine-image-quality",
			wantResolution: "2k",
		},
		{
			name:           "lowercase 1k passes through",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", ImageSize: "1k"},
			wantModel:      "grok-imagine-image",
			wantResolution: "1k",
		},
		{
			name:           "4K silently dropped on grok-imagine",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", ImageSize: "4K"},
			wantModel:      "grok-imagine-image",
			wantResolution: "",
		},
		{
			name:           "image-size ignored for non-imagine model",
			opts:           models.GenerateOptions{Model: "some-non-imagine-model", ImageSize: "1K"},
			wantModel:      "some-non-imagine-model",
			wantResolution: "",
		},
		{
			name:           "aspect-ratio set for imagine model",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", AspectRatio: "16:9"},
			wantModel:      "grok-imagine-image",
			wantResolution: "",
			wantAspect:     "16:9",
		},
		{
			name:           "aspect-ratio auto accepted",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", AspectRatio: "auto"},
			wantModel:      "grok-imagine-image",
			wantAspect:     "auto",
		},
		{
			name:           "aspect-ratio 19.5:9 accepted",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", AspectRatio: "19.5:9"},
			wantModel:      "grok-imagine-image",
			wantAspect:     "19.5:9",
		},
		{
			name:           "aspect-ratio 2:1 accepted",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", AspectRatio: "2:1"},
			wantModel:      "grok-imagine-image",
			wantAspect:     "2:1",
		},
		{
			name:           "invalid aspect-ratio dropped",
			opts:           models.GenerateOptions{Model: "grok-imagine-image", AspectRatio: "5:4"},
			wantModel:      "grok-imagine-image",
			wantAspect:     "",
		},
		{
			name:           "aspect-ratio dropped for non-imagine model",
			opts:           models.GenerateOptions{Model: "some-non-imagine-model", AspectRatio: "16:9"},
			wantModel:      "some-non-imagine-model",
			wantResolution: "",
			wantAspect:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildGrokImageRequest(samplePrompt, tt.opts)
			assert.Equal(t, tt.wantModel, req.Model, "Model")
			assert.Equal(t, tt.wantResolution, req.Resolution, "Resolution")
			assert.Equal(t, tt.wantAspect, req.AspectRatio, "AspectRatio")
			assert.Equal(t, samplePrompt, req.Prompt, "Prompt")
			assert.Equal(t, "b64_json", req.ResponseFormat, "ResponseFormat")
			assert.Nil(t, req.Image)
			assert.Nil(t, req.Images)
		})
	}
}

func TestBuildGrokImageRequest_NumberOfImages(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero defaults to 1", 0, 1},
		{"explicit 1", 1, 1},
		{"max 10", 10, 10},
		{"over-max ignored, defaults to 1", 11, 1},
		{"negative ignored, defaults to 1", -1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildGrokImageRequest("p", models.GenerateOptions{NumberOfImages: tt.in})
			assert.Equal(t, tt.want, req.N)
		})
	}
}

func TestIsValidGrokAspectRatio(t *testing.T) {
	for _, r := range GrokAspectRatios {
		assert.True(t, isValidGrokAspectRatio(r), r)
	}
	assert.False(t, isValidGrokAspectRatio("5:4"))
	assert.False(t, isValidGrokAspectRatio(""))
	assert.False(t, isValidGrokAspectRatio("21:9"))
	assert.Equal(t, 14, len(GrokAspectRatios))
}

func TestAttachGrokInputImages_Single(t *testing.T) {
	fixture := filepath.Join("..", "..", "test", "fixtures", "test_image.png")
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not available: %v", err)
	}

	req := buildGrokImageRequest("edit me", models.GenerateOptions{Model: "grok-imagine-image"})
	err := attachGrokInputImages(&req, []string{fixture})
	require.NoError(t, err)
	require.NotNil(t, req.Image)
	assert.Nil(t, req.Images)
	assert.Equal(t, "image_url", req.Image.Type)
	assert.True(t, strings.HasPrefix(req.Image.URL, "data:image/png;base64,"))

	// JSON should include singular image, not images array
	body, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"image"`)
	assert.NotContains(t, string(body), `"images"`)
}

func TestAttachGrokInputImages_Multi(t *testing.T) {
	fixture := filepath.Join("..", "..", "test", "fixtures", "test_image.png")
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not available: %v", err)
	}

	req := buildGrokImageRequest("compose", models.GenerateOptions{Model: "grok-imagine-image-quality"})
	err := attachGrokInputImages(&req, []string{fixture, fixture, fixture})
	require.NoError(t, err)
	assert.Nil(t, req.Image)
	require.Len(t, req.Images, 3)
	for _, src := range req.Images {
		assert.Equal(t, "image_url", src.Type)
		assert.True(t, strings.HasPrefix(src.URL, "data:image/png;base64,"))
	}

	body, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"images"`)
	// Singular image field should be omitted when using multi
	assert.NotContains(t, string(body), `"image":`)
}

func TestAttachGrokInputImages_TooMany(t *testing.T) {
	req := buildGrokImageRequest("x", models.GenerateOptions{})
	err := attachGrokInputImages(&req, []string{"a.png", "b.png", "c.png", "d.png"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 3")
}

func TestMaxInputImagesForModel_Grok(t *testing.T) {
	assert.Equal(t, grokMaxInputImages, maxInputImagesForModel("grok-imagine-image"))
	assert.Equal(t, grokMaxInputImages, maxInputImagesForModel("grok-imagine-image-quality"))
}
