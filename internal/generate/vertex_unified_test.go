package generate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/apresai/gimage/pkg/models"
	"google.golang.org/genai"
)

func TestBuildVertexGeminiGenerateContentRequest_Features(t *testing.T) {
	fixture := filepath.Join("..", "..", "test", "fixtures", "test_image.png")
	contents, config, width, height, err := buildVertexGeminiGenerateContentRequest(
		"gemini-3.1-flash-image",
		"a product photo",
		models.GenerateOptions{
			Size:               "1792x1024",
			ImageSize:          "2k",
			ThinkingLevel:      "medium",
			WebSearchGrounding: true,
			InputImages:        []string{fixture},
			OutputFormat:       "jpeg",
			NumberOfImages:     3,
		},
	)
	if err != nil {
		t.Fatalf("buildVertexGeminiGenerateContentRequest returned error: %v", err)
	}
	if width != 1792 || height != 1024 {
		t.Fatalf("dimensions = %dx%d, want 1792x1024", width, height)
	}
	if len(contents) != 1 || len(contents[0].Parts) != 2 {
		t.Fatalf("expected text plus one inline image part, got %#v", contents)
	}
	if contents[0].Parts[1].InlineData == nil || contents[0].Parts[1].InlineData.MIMEType != "image/png" {
		t.Fatalf("expected PNG inline data, got %#v", contents[0].Parts[1].InlineData)
	}
	if config.CandidateCount != 3 {
		t.Fatalf("CandidateCount = %d, want 3", config.CandidateCount)
	}
	if config.ImageConfig == nil {
		t.Fatal("expected ImageConfig")
	}
	if config.ImageConfig.ImageSize != "2K" {
		t.Fatalf("ImageSize = %q, want 2K", config.ImageConfig.ImageSize)
	}
	if config.ImageConfig.AspectRatio != "16:9" {
		t.Fatalf("AspectRatio = %q, want 16:9", config.ImageConfig.AspectRatio)
	}
	if config.ImageConfig.OutputMIMEType != "image/jpeg" {
		t.Fatalf("OutputMIMEType = %q, want image/jpeg", config.ImageConfig.OutputMIMEType)
	}
	if config.ThinkingConfig == nil || config.ThinkingConfig.ThinkingLevel != genai.ThinkingLevelMedium {
		t.Fatalf("ThinkingConfig = %#v, want medium", config.ThinkingConfig)
	}
	if len(config.Tools) != 1 || config.Tools[0].GoogleSearch == nil {
		t.Fatalf("expected Google Search grounding tool, got %#v", config.Tools)
	}
}

func TestToGenAIThinkingLevel(t *testing.T) {
	tests := []struct {
		level string
		want  genai.ThinkingLevel
	}{
		{"minimal", genai.ThinkingLevelMinimal},
		{"MINIMAL", genai.ThinkingLevelMinimal},
		{"low", genai.ThinkingLevelLow},
		{"medium", genai.ThinkingLevelMedium},
		{"high", genai.ThinkingLevelHigh},
		{"", genai.ThinkingLevel("")},
		{"unknown", genai.ThinkingLevel("UNKNOWN")},
	}
	for _, tc := range tests {
		t.Run(tc.level, func(t *testing.T) {
			if got := toGenAIThinkingLevel(tc.level); got != tc.want {
				t.Fatalf("toGenAIThinkingLevel(%q) = %q, want %q", tc.level, got, tc.want)
			}
		})
	}
}

func TestSummarizeFinishReasons(t *testing.T) {
	tests := []struct {
		name    string
		reasons []string
		want    string
	}{
		{"all stop", []string{"STOP", "STOP"}, ""},
		{"empty", []string{"", ""}, ""},
		{"safety block", []string{"SAFETY"}, "SAFETY"},
		{"dedupe", []string{"SAFETY", "SAFETY"}, "SAFETY"},
		{"drop stop keep block", []string{"STOP", "PROHIBITED_CONTENT"}, "PROHIBITED_CONTENT"},
		{"multiple distinct", []string{"SAFETY", "IMAGE_SAFETY"}, "SAFETY,IMAGE_SAFETY"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := summarizeFinishReasons(tc.reasons); got != tc.want {
				t.Fatalf("summarizeFinishReasons(%v) = %q, want %q", tc.reasons, got, tc.want)
			}
		})
	}
}

func TestBuildVertexGeminiGenerateContentRequest_CapViolationErrorsEarly(t *testing.T) {
	_, _, _, _, err := buildVertexGeminiGenerateContentRequest(
		"gemini-2.5-flash-image",
		"x",
		models.GenerateOptions{
			InputImages: []string{"/no/file/a.png", "/no/file/b.png", "/no/file/c.png", "/no/file/d.png"},
		},
	)
	if err == nil {
		t.Fatal("expected cap-violation error, got nil")
	}
	if !strings.Contains(err.Error(), "supports at most 3 input images") {
		t.Fatalf("expected cap error, got %v", err)
	}
}
