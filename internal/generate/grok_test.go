package generate

import (
	"testing"

	"github.com/apresai/gimage/pkg/models"
	"github.com/stretchr/testify/assert"
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
			name:           "deprecated -pro alias resolves to quality",
			opts:           models.GenerateOptions{Model: "grok-imagine-pro"},
			wantModel:      "grok-imagine-image-quality",
			wantResolution: "",
		},
		{
			name:           "deprecated long -pro alias resolves to quality",
			opts:           models.GenerateOptions{Model: "grok-imagine-image-pro"},
			wantModel:      "grok-imagine-image-quality",
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
