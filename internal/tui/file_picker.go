// Package tui provides Terminal User Interface components for gimage.
package tui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

// FileInfo represents information about a file in the file picker.
type FileInfo struct {
	Name      string // File name
	Path      string // Full path to file
	Size      int64  // Size in bytes
	IsImage   bool   // Whether it's an image file
	Width     int    // Image width (0 if not an image or not loaded)
	Height    int    // Image height (0 if not an image or not loaded)
	Extension string // File extension (e.g., ".png", ".jpg")
}

// FilePicker provides functionality to browse and select files.
type FilePicker struct {
	directory string
	files     []FileInfo
	filter    []string // File extensions to filter (e.g., [".png", ".jpg"])
}

// NewFilePicker creates a new file picker for the specified directory.
// If directory is empty, uses current working directory.
func NewFilePicker(directory string) (*FilePicker, error) {
	if directory == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		directory = cwd
	}

	// Ensure directory exists
	info, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory %s: %w", directory, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", directory)
	}

	fp := &FilePicker{
		directory: directory,
		files:     []FileInfo{},
	}

	return fp, nil
}

// SetDirectory changes the current directory and refreshes the file list.
func (fp *FilePicker) SetDirectory(directory string) error {
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("failed to access directory %s: %w", directory, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", directory)
	}

	fp.directory = directory
	return fp.Refresh()
}

// SetFilter sets file extensions to filter.
// Extensions should include the dot (e.g., [".png", ".jpg"]).
// Pass nil or empty slice to show all files.
func (fp *FilePicker) SetFilter(extensions []string) {
	fp.filter = make([]string, len(extensions))
	for i, ext := range extensions {
		// Normalize to lowercase with leading dot
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		fp.filter[i] = ext
	}
}

// Refresh re-scans the directory and updates the file list.
func (fp *FilePicker) Refresh() error {
	entries, err := os.ReadDir(fp.directory)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	fp.files = []FileInfo{}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(fp.directory, name)
		ext := strings.ToLower(filepath.Ext(name))

		// Apply filter if set
		if len(fp.filter) > 0 {
			match := false
			for _, filterExt := range fp.filter {
				if ext == filterExt {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		fileInfo := FileInfo{
			Name:      name,
			Path:      path,
			Size:      info.Size(),
			Extension: ext,
			IsImage:   isImageExtension(ext),
		}

		// Try to get image dimensions if it's an image
		if fileInfo.IsImage {
			width, height := getImageDimensions(path)
			fileInfo.Width = width
			fileInfo.Height = height
		}

		fp.files = append(fp.files, fileInfo)
	}

	// Sort files by name
	sort.Slice(fp.files, func(i, j int) bool {
		return fp.files[i].Name < fp.files[j].Name
	})

	return nil
}

// GetDirectory returns the current directory path.
func (fp *FilePicker) GetDirectory() string {
	return fp.directory
}

// GetFile returns FileInfo for a specific index.
// Returns error if index is out of bounds.
func (fp *FilePicker) GetFile(index int) (FileInfo, error) {
	if index < 0 || index >= len(fp.files) {
		return FileInfo{}, fmt.Errorf("index %d out of bounds (0-%d)", index, len(fp.files)-1)
	}
	return fp.files[index], nil
}

// Count returns the number of files in the current list.
func (fp *FilePicker) Count() int {
	return len(fp.files)
}

// isImageExtension checks if a file extension is a known image format.
func isImageExtension(ext string) bool {
	ext = strings.ToLower(ext)
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tiff", ".tif", ".webp":
		return true
	default:
		return false
	}
}

// getImageDimensions tries to get image dimensions without fully decoding.
// Returns (0, 0) if the file can't be read or isn't an image.
func getImageDimensions(path string) (width, height int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	// Decode config (fast, doesn't decode full image)
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}

	return config.Width, config.Height
}

// FormatFileSize formats a file size in bytes to human-readable format.
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatImageInfo formats image dimensions to a readable string.
func FormatImageInfo(file FileInfo) string {
	if !file.IsImage {
		return FormatFileSize(file.Size)
	}
	if file.Width > 0 && file.Height > 0 {
		return fmt.Sprintf("%dx%d, %s", file.Width, file.Height, FormatFileSize(file.Size))
	}
	return FormatFileSize(file.Size)
}

// ListFiles returns a formatted string list of files for display.
// Each line includes the file name and metadata.
func (fp *FilePicker) ListFiles() []string {
	lines := make([]string, len(fp.files))
	for i, file := range fp.files {
		lines[i] = fmt.Sprintf("%s (%s)", file.Name, FormatImageInfo(file))
	}
	return lines
}
