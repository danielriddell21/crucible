package record

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

// FromRGBA wraps a raw RGBA framebuffer (the software renderers' native
// output) as an image without copying.
func FromRGBA(fb []byte, w, h int) *image.RGBA {
	return &image.RGBA{Pix: fb, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
}

// SavePNG writes img as a PNG at path — the family's screenshot key.
func SavePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("record: create %q: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("record: encode %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("record: close %q: %w", path, err)
	}
	return nil
}
