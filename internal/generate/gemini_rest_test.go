package generate

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apresai/gimage/pkg/models"
)

func TestMaxInputImagesForModel(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"gemini-2.5-flash-image", geminiFlash25MaxRefImages},
		{"gemini-3-pro-image", geminiPro3MaxRefImages},
		{"gemini-3.1-flash-image", geminiFlash31MaxRefImages},
		{"unknown-model", geminiFlash25MaxRefImages}, // conservative fallback
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := maxInputImagesForModel(tt.model); got != tt.want {
				t.Errorf("maxInputImagesForModel(%q) = %d, want %d", tt.model, got, tt.want)
			}
		})
	}
}

func TestReadInputImageAsInlineData_FixturePNG(t *testing.T) {
	fixture := filepath.Join("..", "..", "test", "fixtures", "test_image.png")
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	part, err := readInputImageAsInlineData(fixture)
	if err != nil {
		t.Fatalf("readInputImageAsInlineData(%q) returned error: %v", fixture, err)
	}
	if part.InlineData == nil {
		t.Fatal("expected InlineData to be set, got nil")
	}
	if part.InlineData.MimeType != "image/png" {
		t.Errorf("expected MimeType image/png, got %q", part.InlineData.MimeType)
	}
	if part.InlineData.Data == "" {
		t.Error("expected non-empty base64 Data")
	}
	decoded, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	original, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("could not read fixture for comparison: %v", err)
	}
	if len(decoded) != len(original) {
		t.Errorf("decoded length %d != fixture length %d", len(decoded), len(original))
	}
}

func TestReadInputImageAsInlineData_MissingFile(t *testing.T) {
	_, err := readInputImageAsInlineData("/nonexistent/path/that/should/not/exist.png")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !os.IsNotExist(err) && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("expected fs missing-file error, got: %v", err)
	}
}

func TestReadInputImageData_RemoteURLRejected(t *testing.T) {
	_, _, err := readInputImageData("https://example.com/photo.png")
	if err == nil {
		t.Fatal("expected error for remote URL, got nil")
	}
	if !strings.Contains(err.Error(), "only supported for Grok") {
		t.Errorf("expected Grok-only URL message, got: %v", err)
	}
}

func TestReadInputImageAsInlineData_TooLarge(t *testing.T) {
	// Sparse-allocate a file just over maxInputImageBytes (20 MB) — Truncate
	// reserves logical size without writing actual bytes, so the test stays fast
	// and disk-cheap. The size check uses os.Stat which reads the inode, not
	// the data, so a sparse file is sufficient.
	tmp, err := os.CreateTemp(t.TempDir(), "huge-*.png")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer tmp.Close()
	if err := tmp.Truncate(maxInputImageBytes + 1); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}
	_, err = readInputImageAsInlineData(tmp.Name())
	if err == nil {
		t.Fatal("expected size-limit error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("expected error to mention size limit; got: %v", err)
	}
}

func TestReadInputImageAsInlineData_UnsupportedMIME(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "fake-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	if _, err := tmp.WriteString("This is plainly not an image, it is prose."); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	tmp.Close()
	_, err = readInputImageAsInlineData(tmp.Name())
	if err == nil {
		t.Fatal("expected MIME-rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported MIME type") {
		t.Errorf("expected unsupported-MIME error, got: %v", err)
	}
}

func TestGeminiGenerationConfig_ThinkingMarshaling(t *testing.T) {
	cfg := &geminiGenerationConfig{
		ResponseModalities: []string{"TEXT", "IMAGE"},
		ThinkingConfig:     &geminiThinkingConfig{ThinkingLevel: "medium"},
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(out), `"thinkingConfig":{"thinkingLevel":"medium"}`) {
		t.Errorf("expected thinkingConfig.thinkingLevel=medium in JSON; got: %s", out)
	}

	cfgNoThink := &geminiGenerationConfig{ResponseModalities: []string{"IMAGE"}}
	outNoThink, err := json.Marshal(cfgNoThink)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(outNoThink), "thinkingConfig") {
		t.Errorf("did not expect thinkingConfig key when nil; got: %s", outNoThink)
	}
}

func TestGeminiTool_GoogleSearchMarshaling(t *testing.T) {
	tools := []geminiTool{{GoogleSearch: &geminiGoogleSearch{}}}
	out, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(out) != `[{"google_search":{}}]` {
		t.Errorf("unexpected tool JSON; got: %s", out)
	}

	out2, err := json.Marshal([]geminiTool{})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(out2) != `[]` {
		t.Errorf("expected [] for empty tools; got: %s", out2)
	}
}

func TestGeminiRequest_ToolsOmittedWhenEmpty(t *testing.T) {
	req := geminiGenerateContentRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: "hi"}}}},
	}
	out, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(out), "tools") {
		t.Errorf("did not expect tools field in payload when empty; got: %s", out)
	}
}

// newTestGeminiClient constructs a client suitable for unit tests of the
// request-building path. It carries a real VerboseLogger so c.log.Debug calls
// inside buildGenerateContentRequest don't nil-panic.
func newTestGeminiClient(t *testing.T) *GeminiRESTClient {
	t.Helper()
	c, err := NewGeminiRESTClient("test-key-not-used-by-build-path")
	if err != nil {
		t.Fatalf("NewGeminiRESTClient failed: %v", err)
	}
	return c
}

// TestBuildGenerateContentRequest_AdvancedActivatesThinkingAndGrounding verifies
// that on Gemini 3+ models, ThinkingLevel and WebSearchGrounding produce the
// expected request struct fields (thinkingConfig + tools). This is the positive
// branch coverage that the bare JSON-marshal tests don't exercise.
func TestBuildGenerateContentRequest_AdvancedActivatesThinkingAndGrounding(t *testing.T) {
	c := newTestGeminiClient(t)
	req, err := c.buildGenerateContentRequest(
		"gemini-3-pro-image",
		"a sunset",
		models.GenerateOptions{
			ThinkingLevel:      "medium",
			WebSearchGrounding: true,
		},
	)
	if err != nil {
		t.Fatalf("buildGenerateContentRequest returned error: %v", err)
	}
	if req.GenerationConfig == nil || req.GenerationConfig.ThinkingConfig == nil {
		t.Fatal("expected ThinkingConfig to be set on Gemini 3 Pro")
	}
	if req.GenerationConfig.ThinkingConfig.ThinkingLevel != "medium" {
		t.Errorf("expected thinkingLevel=medium, got %q", req.GenerationConfig.ThinkingConfig.ThinkingLevel)
	}
	if len(req.Tools) != 1 || req.Tools[0].GoogleSearch == nil {
		t.Errorf("expected exactly one tool with GoogleSearch set, got %#v", req.Tools)
	}
	if len(req.Contents) != 1 || req.Contents[0].Role != "user" {
		t.Errorf("expected one content with role=user, got %#v", req.Contents)
	}
}

// TestBuildGenerateContentRequest_NonAdvancedSilentlyDropsAdvancedFields is the
// regression guard for the gating behavior the diff promises: on Gemini 2.5
// Flash, both ThinkingLevel and WebSearchGrounding must be silently dropped —
// the resulting request has no thinkingConfig and no tools field.
func TestBuildGenerateContentRequest_NonAdvancedSilentlyDropsAdvancedFields(t *testing.T) {
	c := newTestGeminiClient(t)
	req, err := c.buildGenerateContentRequest(
		"gemini-2.5-flash-image",
		"a sunset",
		models.GenerateOptions{
			ThinkingLevel:      "high",
			WebSearchGrounding: true,
		},
	)
	if err != nil {
		t.Fatalf("buildGenerateContentRequest returned error: %v", err)
	}
	if req.GenerationConfig != nil && req.GenerationConfig.ThinkingConfig != nil {
		t.Errorf("did not expect ThinkingConfig on Gemini 2.5 Flash, got %#v", req.GenerationConfig.ThinkingConfig)
	}
	if len(req.Tools) != 0 {
		t.Errorf("did not expect tools on Gemini 2.5 Flash, got %#v", req.Tools)
	}
	// Also confirm the marshaled wire payload contains neither field.
	out, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(out), "thinkingConfig") {
		t.Errorf("Gemini 2.5 payload should not contain thinkingConfig; got: %s", out)
	}
	if strings.Contains(string(out), "tools") {
		t.Errorf("Gemini 2.5 payload should not contain tools; got: %s", out)
	}
}

// TestBuildGenerateContentRequest_CapViolationErrorsEarly verifies that passing
// more reference images than a model supports surfaces an actionable error
// BEFORE any I/O — no file reads, no HTTP. This is the safety net for the
// per-model caps in maxInputImagesForModel.
func TestBuildGenerateContentRequest_CapViolationErrorsEarly(t *testing.T) {
	c := newTestGeminiClient(t)
	// gemini-2.5-flash-image cap is 3. Pass 4 to trip it.
	// Use bogus paths — the error must fire BEFORE we attempt to read them,
	// so the paths never need to exist on disk.
	_, err := c.buildGenerateContentRequest(
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
		t.Errorf("expected cap-violation message, got: %v", err)
	}
}

func TestBuildGenerateContentRequest_LiteCoercesThinkingAndDropsGrounding(t *testing.T) {
	c := newTestGeminiClient(t)
	req, err := c.buildGenerateContentRequest(
		"gemini-3.1-flash-lite-image",
		"a sunset",
		models.GenerateOptions{
			ThinkingLevel:      "medium",
			WebSearchGrounding: true,
			ImageSize:          "2K",
		},
	)
	if err != nil {
		t.Fatalf("buildGenerateContentRequest returned error: %v", err)
	}
	if req.GenerationConfig == nil || req.GenerationConfig.ThinkingConfig == nil {
		t.Fatal("expected ThinkingConfig on Flash Lite")
	}
	if req.GenerationConfig.ThinkingConfig.ThinkingLevel != "high" {
		t.Errorf("expected thinkingLevel=high (coerced from medium), got %q", req.GenerationConfig.ThinkingConfig.ThinkingLevel)
	}
	if len(req.Tools) != 0 {
		t.Errorf("did not expect grounding tools on Flash Lite, got %#v", req.Tools)
	}
	if req.GenerationConfig.ImageConfig != nil && req.GenerationConfig.ImageConfig.ImageSize == "2K" {
		t.Errorf("Flash Lite must not send imageSize 2K, got %#v", req.GenerationConfig.ImageConfig)
	}

	req, err = c.buildGenerateContentRequest(
		"gemini-3.1-flash-lite-image",
		"a sunset",
		models.GenerateOptions{ThinkingLevel: "low", ImageSize: "1K"},
	)
	if err != nil {
		t.Fatalf("buildGenerateContentRequest returned error: %v", err)
	}
	if req.GenerationConfig.ThinkingConfig.ThinkingLevel != "minimal" {
		t.Errorf("expected thinkingLevel=minimal (coerced from low), got %q", req.GenerationConfig.ThinkingConfig.ThinkingLevel)
	}
	if req.GenerationConfig.ImageConfig == nil || req.GenerationConfig.ImageConfig.ImageSize != "1K" {
		t.Errorf("expected imageSize 1K on Flash Lite, got %#v", req.GenerationConfig.ImageConfig)
	}
}
