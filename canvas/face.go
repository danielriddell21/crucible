package canvas

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// TextFace draws s with its baseline at (x, y) in a caller-supplied face,
// returning the advance in pixels. It is the escape hatch from the built-in
// 7x13 face for front-ends with their own typography — a chess board's piece
// glyphs, a title screen's display face — so they can compose headlessly
// without giving up their look.
//
// A nil face falls back to the built-in one.
func (c *Canvas) TextFace(x, y int, s string, col color.RGBA, face font.Face) int {
	if face == nil {
		c.Text(x, y, s, col)
		return len(s) * GlyphWidth
	}
	d := &font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	start := d.Dot.X
	d.DrawString(s)
	return (d.Dot.X - start).Ceil()
}

// MeasureFace returns the pixel advance s would take in face, so a caller can
// centre or wrap text before drawing it. A nil face measures the built-in one.
func MeasureFace(s string, face font.Face) int {
	if face == nil {
		return len(s) * GlyphWidth
	}
	return font.MeasureString(face, s).Ceil()
}
