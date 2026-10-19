package imaging

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpsEdgeCases(t *testing.T) {
	t.Run("nil and non-positive bounds", func(t *testing.T) {
		assert.True(t, Clone(nil).Bounds().Empty())
		assert.True(t, Resize(nil, 100, 100).Bounds().Empty())
		assert.True(t, Fit(nil, 100, 100).Bounds().Empty())
		assert.True(t, Fill(nil, 100, 100).Bounds().Empty())
		assert.True(t, Crop(nil, image.Rect(0, 0, 10, 10)).Bounds().Empty())

		src := image.NewNRGBA(image.Rect(0, 0, 50, 50))
		assert.True(t, Resize(src, 0, 50).Bounds().Empty())
		assert.True(t, Resize(src, 50, -1).Bounds().Empty())
		assert.True(t, Fit(src, 0, 50).Bounds().Empty())
		assert.True(t, Fill(src, -5, 50).Bounds().Empty())
		assert.True(t, Crop(src, image.Rect(100, 100, 120, 120)).Bounds().Empty())
	})

	t.Run("Save nil image and unsupported extension", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.Error(t, Save(nil, filepath.Join(tmpDir, "out.png")))

		src := image.NewNRGBA(image.Rect(0, 0, 10, 10))
		assert.Error(t, Save(src, filepath.Join(tmpDir, "noext")))
		assert.Error(t, Save(src, filepath.Join(tmpDir, "out.xyz")))
	})

	t.Run("Format round-trips including WebP, GIF, TIFF, BMP, JPEG with transparency", func(t *testing.T) {
		tmpDir := t.TempDir()
		src := image.NewNRGBA(image.Rect(0, 0, 32, 24))
		for y := 0; y < 24; y++ {
			for x := 0; x < 32; x++ {
				src.SetNRGBA(x, y, color.NRGBA{R: 200, G: 100, B: 50, A: 128})
			}
		}

		for _, ext := range []string{".png", ".jpg", ".webp", ".gif", ".tiff", ".bmp"} {
			p := filepath.Join(tmpDir, "sub", "test"+ext)
			require.NoError(t, Save(src, p), "Save(%s)", ext)

			decoded, err := Open(p)
			require.NoError(t, err, "Open(%s)", ext)
			assert.Equal(t, 32, decoded.Bounds().Dx(), "width for %s", ext)
			assert.Equal(t, 24, decoded.Bounds().Dy(), "height for %s", ext)
		}
	})
}
