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
