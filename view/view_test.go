package view_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/view"
)

func TestRoundTrip(t *testing.T) {
	c := view.New(800, 600)
	c.Follow(10, 20)
	c.Zoom = 2
	sx, sy := c.WorldToScreen(13, 24)
	wx, wy := c.ScreenToWorld(sx, sy)
	if math.Abs(wx-13) > 1e-9 || math.Abs(wy-24) > 1e-9 {
		t.Errorf("round trip changed the point: (13,24) -> (%.4f,%.4f)", wx, wy)
	}
}

func TestCentreMapsToScreenCentre(t *testing.T) {
	c := view.New(800, 600)
	c.Follow(10, 20)
	sx, sy := c.WorldToScreen(10, 20)
	if sx != 400 || sy != 300 {
		t.Errorf("focus should map to screen centre, got (%.1f,%.1f)", sx, sy)
	}
}

func TestFitBounds(t *testing.T) {
	c := view.New(800, 400)
	c.FitBounds(0, 0, 100, 100, 0) // 100x100 world into 800x400
	if c.X != 50 || c.Y != 50 {
		t.Errorf("centre = (%.1f,%.1f), want (50,50)", c.X, c.Y)
	}
	if math.Abs(c.Zoom-4) > 1e-9 { // limited by height: 400/100
		t.Errorf("zoom = %.3f, want 4 (height-limited)", c.Zoom)
	}
	c.FitBounds(0, 0, 0, 0, 0) // degenerate
	if c.Zoom != 1 {
		t.Errorf("degenerate bounds should reset zoom to 1, got %.3f", c.Zoom)
	}
}
