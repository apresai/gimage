package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFilePicker(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		wantError bool
	}{
		{
			name:      "empty directory uses current",
			directory: "",
			wantError: false,
		},
		{
			name:      "valid directory",
			directory: os.TempDir(),
			wantError: false,
		},
		{
			name:      "non-existent directory",
			directory: "/path/that/does/not/exist",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp, err := NewFilePicker(tt.directory)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if fp == nil {
				t.Errorf("Expected FilePicker but got nil")
			}
		})
	}
}

func TestFilePickerSetFilter(t *testing.T) {
	fp, err := NewFilePicker(os.TempDir())
	if err != nil {
		t.Fatalf("Failed to create FilePicker: %v", err)
	}

	tests := []struct {
		name       string
		extensions []string
		want       []string
	}{
		{
			name:       "normalize extensions",
			extensions: []string{"png", ".jpg", "JPEG"},
			want:       []string{".png", ".jpg", ".jpeg"},
		},
		{
			name:       "empty filter",
			extensions: []string{},
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp.SetFilter(tt.extensions)
			if len(fp.filter) != len(tt.want) {
				t.Errorf("Expected %d filters, got %d", len(tt.want), len(fp.filter))
				return
			}
			for i, ext := range tt.want {
				if fp.filter[i] != ext {
					t.Errorf("Expected filter[%d] = %s, got %s", i, ext, fp.filter[i])
				}
			}
		})
	}
}

func TestIsImageExtension(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".png", true},
		{".PNG", true},
		{".jpg", true},
		{".jpeg", true},
		{".gif", true},
		{".bmp", true},
		{".tiff", true},
		{".tif", true},
		{".webp", true},
		{".txt", false},
		{".pdf", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := isImageExtension(tt.ext)
			if got != tt.want {
				t.Errorf("isImageExtension(%s) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatFileSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatFileSize(%d) = %s, want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatImageInfo(t *testing.T) {
	tests := []struct {
		name string
		file FileInfo
		want string
	}{
		{
			name: "non-image file",
			file: FileInfo{
				Name:    "test.txt",
				Size:    1024,
				IsImage: false,
			},
			want: "1.0 KB",
		},
		{
			name: "image with dimensions",
			file: FileInfo{
				Name:    "test.png",
				Size:    2048,
				IsImage: true,
				Width:   800,
				Height:  600,
			},
			want: "800x600, 2.0 KB",
		},
		{
			name: "image without dimensions",
			file: FileInfo{
				Name:    "test.jpg",
				Size:    3072,
				IsImage: true,
				Width:   0,
				Height:  0,
			},
			want: "3.0 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatImageInfo(tt.file)
			if got != tt.want {
				t.Errorf("FormatImageInfo() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFilePickerRefresh(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"test1.txt", "hello"},
		{"test2.png", "fake png"},
		{"test3.jpg", "fake jpg"},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, tf.name)
		if err := os.WriteFile(path, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	fp, err := NewFilePicker(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create FilePicker: %v", err)
	}

	// Test without filter
	if err := fp.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	if fp.Count() != 3 {
		t.Errorf("Expected 3 files, got %d", fp.Count())
	}

	// Test with filter
	fp.SetFilter([]string{".png", ".jpg"})
	if err := fp.Refresh(); err != nil {
		t.Fatalf("Refresh with filter failed: %v", err)
	}

	if fp.Count() != 2 {
		t.Errorf("Expected 2 files with filter, got %d", fp.Count())
	}

	// Files should be sorted by name
	file0, err := fp.GetFile(0)
	if err != nil {
		t.Fatalf("GetFile(0) failed: %v", err)
	}
	file1, err := fp.GetFile(1)
	if err != nil {
		t.Fatalf("GetFile(1) failed: %v", err)
	}
	if file0.Name != "test2.png" {
		t.Errorf("Expected first file to be test2.png, got %s", file0.Name)
	}
	if file1.Name != "test3.jpg" {
		t.Errorf("Expected second file to be test3.jpg, got %s", file1.Name)
	}
}

func TestFilePickerGetFile(t *testing.T) {
	tmpDir := t.TempDir()
	fp, err := NewFilePicker(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create FilePicker: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := fp.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	// Test valid index
	file, err := fp.GetFile(0)
	if err != nil {
		t.Errorf("GetFile(0) returned error: %v", err)
	}
	if file.Name != "test.txt" {
		t.Errorf("Expected file name 'test.txt', got '%s'", file.Name)
	}

	// Test invalid indices
	invalidIndices := []int{-1, 1, 100}
	for _, idx := range invalidIndices {
		_, err := fp.GetFile(idx)
		if err == nil {
			t.Errorf("GetFile(%d) should return error for out of bounds index", idx)
		}
	}
}

func TestFilePickerListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fp, err := NewFilePicker(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create FilePicker: %v", err)
	}

	if err := fp.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	lines := fp.ListFiles()
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	// Line should contain filename and size info
	if len(lines[0]) == 0 {
		t.Errorf("Expected non-empty line")
	}
}
