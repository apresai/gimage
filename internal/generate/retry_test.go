package generate

import (
	"context"
	"testing"
)

func TestExtractFormatFromMimeType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		want     string
	}{
		{
			name:     "image/png",
			mimeType: "image/png",
			want:     "png",
		},
		{
			name:     "image/jpeg",
			mimeType: "image/jpeg",
			want:     "jpg",
		},
		{
			name:     "image/jpg",
			mimeType: "image/jpg",
			want:     "jpg",
		},
		{
			name:     "image/webp",
			mimeType: "image/webp",
			want:     "webp",
		},
		{
			name:     "image/gif",
			mimeType: "image/gif",
			want:     "gif",
		},
		{
			name:     "unknown type",
			mimeType: "application/octet-stream",
			want:     "png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractFormatFromMimeType(tt.mimeType); got != tt.want {
				t.Errorf("extractFormatFromMimeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "deadline exceeded error",
			err:  context.DeadlineExceeded,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}
