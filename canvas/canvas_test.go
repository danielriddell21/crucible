package canvas_test

import (
	"image/color"
	"testing"

	"github.com/danielriddell21/crucible/canvas"
)

func pixel(c *canvas.Canvas, x, y int) [4]byte {
	w, _ := c.Size()
	i := (y*w + x) * 4
	p := c.Pixels()
	return [4]byte{p[i], p[i+1], p[i+2], p[i+3]}
}

func TestFill(t *testing.T) {
	c := canvas.New(4, 3)
	c.Fill(color.RGBA{R: 10, G: 20, B: 30, A: 255})
	if got := pixel(c, 3, 2); got != [4]byte{10, 20, 30, 255} {
		t.Errorf("pixel = %v", got)
	}
}

func TestRectClipped(t *testing.T) {
	c := canvas.New(4, 4)
	red := color.RGBA{R: 200, A: 255}
	c.Rect(-2, -2, 4, 4, red) // half off-canvas; must not panic
	if got := pixel(c, 1, 1); got != [4]byte{200, 0, 0, 255} {
		t.Errorf("inside pixel = %v", got)
	}
	if got := pixel(c, 2, 2); got != [4]byte{0, 0, 0, 0} {
		t.Errorf("outside pixel = %v", got)
	}
}

func TestDimFrom(t *testing.T) {
	c := canvas.New(1, 1)
	c.DimFrom([]byte{100, 200, 50, 255}, 0.5)
	if got := pixel(c, 0, 0); got != [4]byte{50, 100, 25, 255} {
		t.Errorf("dimmed pixel = %v", got)
	}
}

func TestTextMarksPixels(t *testing.T) {
	c := canvas.New(40, 20)
	c.Text(2, 12, "A", color.RGBA{R: 255, G: 255, B: 255, A: 255})
	sum := 0
	for _, b := range c.Pixels() {
		sum += int(b)
	}
	if sum == 0 {
		t.Fatal("Text drew nothing")
	}
}

func TestTextCenteredDrawsShadow(t *testing.T) {
	c := canvas.New(40, 20)
	c.TextCentered(12, "AB", color.RGBA{R: 255, A: 255})
	// The shadow is pure black on a transparent canvas: look for a pixel with
	// alpha set but no red.
	p := c.Pixels()
	shadow := false
	for i := 0; i < len(p); i += 4 {
		if p[i+3] == 255 && p[i] == 0 {
			shadow = true
			break
		}
	}
	if !shadow {
		t.Fatal("no shadow pixels found")
	}
}
