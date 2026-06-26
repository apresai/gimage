package generate

import (
	"strings"
	"testing"
)

func TestEnhancePrompt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty prompt",
			input: "",
			want:  "",
		},
		{
			name:  "short prompt gets enhanced",
			input: "a cat",
			want:  "a cat, high quality, detailed, professional, sharp focus, well composed",
		},
		{
			name:  "long prompt unchanged",
			input: strings.Repeat("a ", 60),
			want:  strings.TrimSpace(strings.Repeat("a ", 60)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnhancePrompt(tt.input)
			if got != tt.want {
				t.Errorf("EnhancePrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnhancePromptWithStyle(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		style  string
		want   string
	}{
		{
			name:   "photorealistic style",
			prompt: "a landscape",
			style:  "photorealistic",
			want:   "a landscape, highly detailed photorealistic image, professional photography, 8k resolution, realistic lighting",
		},
		{
			name:   "anime style",
			prompt: "a character",
			style:  "anime",
			want:   "a character, anime style, manga art, vibrant colors, detailed character design",
		},
		{
			name:   "unknown style",
			prompt: "a cat",
			style:  "unknown-style",
			want:   "a cat, high quality, detailed, professional, sharp focus, well composed",
		},
		{
			name:   "empty prompt",
			prompt: "",
			style:  "photorealistic",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnhancePromptWithStyle(tt.prompt, tt.style)
			if got != tt.want {
				t.Errorf("EnhancePromptWithStyle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{
			name:    "valid prompt",
			prompt:  "a beautiful landscape",
			wantErr: false,
		},
		{
			name:    "empty prompt",
			prompt:  "",
			wantErr: true,
		},
		{
			name:    "too short prompt",
			prompt:  "ab",
			wantErr: true,
		},
		{
			name:    "too long prompt",
			prompt:  strings.Repeat("a", 2001),
			wantErr: true,
		},
		{
			name:    "minimum valid length",
			prompt:  "abc",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name         string
		prompt       string
		wantMinWords int
		wantMaxWords int
	}{
		{
			name:         "simple prompt",
			prompt:       "a beautiful sunset over the ocean",
			wantMinWords: 2,
			wantMaxWords: 10,
		},
		{
			name:         "prompt with punctuation",
			prompt:       "a cat, sitting on a mat, looking at the camera.",
			wantMinWords: 2,
			wantMaxWords: 10,
		},
		{
			name:         "long prompt",
			prompt:       strings.Repeat("word ", 20),
			wantMinWords: 1,
			wantMaxWords: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.prompt)
			if len(got) < tt.wantMinWords || len(got) > tt.wantMaxWords {
				t.Errorf("ExtractKeywords() returned %d words, want between %d and %d",
					len(got), tt.wantMinWords, tt.wantMaxWords)
			}
		})
	}
}

func TestTruncatePrompt(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		maxLength int
		wantLen   int
	}{
		{
			name:      "no truncation needed",
			prompt:    "short prompt",
			maxLength: 100,
			wantLen:   12,
		},
		{
			name:      "truncation with word boundary",
			prompt:    "this is a very long prompt that needs truncation",
			maxLength: 20,
			wantLen:   20, // "this is a very..." = 19 chars
		},
		{
			name:      "exact length",
			prompt:    "exact",
			maxLength: 5,
			wantLen:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncatePrompt(tt.prompt, tt.maxLength)
			if len(got) > tt.maxLength+3 { // +3 for "..."
				t.Errorf("TruncatePrompt() length = %d, want <= %d", len(got), tt.maxLength+3)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "multiple spaces",
			input: "hello    world",
			want:  "hello world",
		},
		{
			name:  "line breaks",
			input: "hello\r\nworld",
			want:  "hello\nworld",
		},
		{
			name:  "mixed whitespace",
			input: "  hello  \n  world  ",
			want:  "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("normalizeWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatPromptForDisplay(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		maxLength int
		wantLen   int
	}{
		{
			name:      "no truncation",
			prompt:    "short",
			maxLength: 100,
			wantLen:   5,
		},
		{
			name:      "with truncation",
			prompt:    strings.Repeat("word ", 50),
			maxLength: 50,
			wantLen:   53, // truncated + "..."
		},
		{
			name:      "no max length",
			prompt:    "any length allowed",
			maxLength: 0,
			wantLen:   18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPromptForDisplay(tt.prompt, tt.maxLength)
			if tt.maxLength > 0 && len(got) > tt.maxLength+3 {
				t.Errorf("FormatPromptForDisplay() length = %d, want <= %d", len(got), tt.maxLength+3)
			}
		})
	}
}
