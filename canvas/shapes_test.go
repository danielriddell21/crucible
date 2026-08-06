package canvas_test

import (
	"image/color"
	"testing"

	"github.com/danielriddell21/crucible/canvas"
)

var red = color.RGBA{R: 255, A: 255}

// alphaAt reports the red channel at a pixel, which for the red test colour
// doubles as how much of the shape covers it.
func alphaAt(c *canvas.Canvas, x, y int) byte {
	w, _ := c.Size()
	return c.Pixels()[(y*w+x)*4]
}

func TestPolygonFillsInterior(t *testing.T) {
	c := canvas.New(20, 20)
	// A square covering the middle of the canvas.
	c.Polygon([][2]float64{{5, 5}, {15, 5}, {15, 15}, {5, 15}}, red)

	if got := alphaAt(c, 10, 10); got != 255 {
		t.Errorf("centre = %d, want fully covered", got)
	}
	if got := alphaAt(c, 1, 1); got != 0 {
		t.Errorf("outside = %d, want untouched", got)
	}
}

func TestPolygonNeedsThreePoints(t *testing.T) {
	c := canvas.New(8, 8)
	c.Polygon([][2]float64{{0, 0}, {7, 7}}, red)
	for y := range 8 {
		for x := range 8 {
			if alphaAt(c, x, y) != 0 {
				t.Fatalf("a two-point polygon should draw nothing, but (%d,%d) was painted", x, y)
			}
		}
	}
}

func TestCircleFillsAndIsRound(t *testing.T) {
	c := canvas.New(40, 40)
	c.Circle(20, 20, 12, red)

	if got := alphaAt(c, 20, 20); got != 255 {
		t.Errorf("centre = %d, want fully covered", got)
	}
	// Just outside the radius on both axes stays clear: the shape is a disc,
	// not its bounding box.
	if got := alphaAt(c, 20+14, 20); got != 0 {
		t.Errorf("beyond the radius = %d, want untouched", got)
	}
	if got := alphaAt(c, 20+11, 20+11); got != 0 {
		t.Errorf("the bounding-box corner = %d, want untouched (a box, not a disc)", got)
	}
}

func TestCircleAntiAliasesItsEdge(t *testing.T) {
	c := canvas.New(40, 40)
	c.Circle(20, 20, 10, red)
	// Somewhere along the rim there must be partial coverage, or the edge is
	// hard-aliased.
	partial := false
	for x := range 40 {
		if a := alphaAt(c, x, 20); a > 0 && a < 255 {
			partial = true
			break
		}
	}
	if !partial {
		t.Error("expected anti-aliased coverage along the circle edge")
	}
}

func TestCircleIgnoresNonPositiveRadius(t *testing.T) {
	c := canvas.New(8, 8)
	c.Circle(4, 4, 0, red)
	if alphaAt(c, 4, 4) != 0 {
		t.Error("a zero radius should draw nothing")
	}
}

func TestLineDrawsBetweenEndpoints(t *testing.T) {
	c := canvas.New(20, 20)
	c.Line(2, 10, 18, 10, 2, red)

	if got := alphaAt(c, 10, 10); got == 0 {
		t.Error("the line's midpoint should be painted")
	}
	if got := alphaAt(c, 10, 2); got != 0 {
		t.Errorf("far from the line = %d, want untouched", got)
	}
}

func TestLineOfZeroLengthDrawsNothing(t *testing.T) {
	c := canvas.New(8, 8)
	c.Line(4, 4, 4, 4, 2, red)
	if alphaAt(c, 4, 4) != 0 {
		t.Error("a zero-length line should draw nothing")
	}
}

func TestShapesClipToTheCanvas(t *testing.T) {
	c := canvas.New(10, 10)
	// Shapes hanging off every edge must not panic or corrupt memory.
	c.Circle(-5, -5, 8, red)
	c.Circle(14, 14, 8, red)
	c.Polygon([][2]float64{{-20, -20}, {-10, -20}, {-10, -10}}, red)
	c.Line(-30, 5, 40, 5, 3, red)

	if got := alphaAt(c, 5, 5); got == 0 {
		t.Error("the line crossing the canvas should still paint it")
	}
}

func TestBlendCompositesOverExistingPixels(t *testing.T) {
	c := canvas.New(10, 10)
	c.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	// Half-opacity white over black lands about halfway.
	c.Blend(0, 0, 10, 10, color.RGBA{R: 255, G: 255, B: 255, A: 128})
	got := alphaAt(c, 5, 5)
	if got < 120 || got > 136 {
		t.Errorf("blended value = %d, want ~128", got)
	}
}

func TestBlendFullyOpaqueReplaces(t *testing.T) {
	c := canvas.New(6, 6)
	c.Fill(color.RGBA{A: 255})
	c.Blend(0, 0, 6, 6, color.RGBA{R: 200, A: 255})
	if got := alphaAt(c, 3, 3); got != 200 {
		t.Errorf("opaque blend = %d, want 200", got)
	}
}

func TestBlendTransparentIsNoOp(t *testing.T) {
	c := canvas.New(6, 6)
	c.Fill(color.RGBA{R: 10, A: 255})
	c.Blend(0, 0, 6, 6, color.RGBA{R: 250})
	if got := alphaAt(c, 3, 3); got != 10 {
		t.Errorf("zero alpha changed the pixel to %d", got)
	}
}

func TestBlendClipsToCanvas(t *testing.T) {
	c := canvas.New(6, 6)
	c.Fill(color.RGBA{A: 255})
	c.Blend(-4, -4, 20, 20, color.RGBA{R: 255, A: 255}) // must not panic
	if got := alphaAt(c, 0, 0); got != 255 {
		t.Errorf("clipped blend = %d, want it to still paint in-bounds pixels", got)
	}
}
