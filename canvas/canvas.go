// Package canvas provides a small software drawing surface for menus and
// text overlays, so a front-end can compose text screens the same way a
// software renderer composes frames: as raw RGBA pixels.
//
// [New] returns a [Canvas] sized to the window. Callers compose a screen
// with [Canvas.Fill] or [Canvas.DimFrom], [Canvas.Rect], [Canvas.Text],
// and [Canvas.TextCentered], then pass [Canvas.Pixels] to the WritePixels
// method of an Ebiten image. The package does not import Ebiten itself, so
// it needs no display.
package canvas

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// GlyphWidth is the advance of the built-in 7x13 face, used to centre text.
const GlyphWidth = 7

// Canvas is a fixed-size RGBA pixel surface.
type Canvas struct {
	img  *image.RGBA
	w, h int
}

// New returns a canvas of the given size in pixels.
func New(w, h int) *Canvas {
	return &Canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), w: w, h: h}
}

// Size returns the canvas width and height in pixels.
func (c *Canvas) Size() (w, h int) { return c.w, c.h }

// Pixels returns the underlying RGBA pixel buffer, suitable for
// (*ebiten.Image).WritePixels.
func (c *Canvas) Pixels() []byte { return c.img.Pix }

// Fill paints the whole canvas with a single opaque colour.
func (c *Canvas) Fill(col color.RGBA) {
	for i := 0; i < len(c.img.Pix); i += 4 {
		c.img.Pix[i], c.img.Pix[i+1], c.img.Pix[i+2], c.img.Pix[i+3] = col.R, col.G, col.B, 255
	}
}

// DimFrom copies another framebuffer scaled toward black by factor k in
// [0, 1], so a paused game shows faintly behind its menu.
func (c *Canvas) DimFrom(fb []byte, k float64) {
	n := min(len(fb), len(c.img.Pix))
	for i := 0; i < n; i += 4 {
		c.img.Pix[i] = uint8(float64(fb[i]) * k)
		c.img.Pix[i+1] = uint8(float64(fb[i+1]) * k)
		c.img.Pix[i+2] = uint8(float64(fb[i+2]) * k)
		c.img.Pix[i+3] = 255
	}
}

// Rect fills an axis-aligned rectangle, clipped to the canvas.
func (c *Canvas) Rect(x, y, w, h int, col color.RGBA) {
	for dy := range h {
		py := y + dy
		if py < 0 || py >= c.h {
			continue
		}
		for dx := range w {
			px := x + dx
			if px < 0 || px >= c.w {
				continue
			}
			i := (py*c.w + px) * 4
			c.img.Pix[i], c.img.Pix[i+1], c.img.Pix[i+2], c.img.Pix[i+3] = col.R, col.G, col.B, 255
		}
	}
}

// Text draws s with its baseline at (x, y) in the built-in 7x13 face.
func (c *Canvas) Text(x, y int, s string, col color.RGBA) {
	d := &font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

// TextCentered draws s horizontally centred at baseline y, with a one-pixel
// drop shadow.
func (c *Canvas) TextCentered(y int, s string, col color.RGBA) {
	x := (c.w - len(s)*GlyphWidth) / 2
	c.Text(x+1, y+1, s, color.RGBA{A: 255}) // drop shadow
	c.Text(x, y, s, col)
}
