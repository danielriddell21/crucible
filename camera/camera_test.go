package camera_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/camera"
)

const eps = 1e-9

func TestScreenToWorldRoundTrip(t *testing.T) {
	c := camera.New()
	c.X, c.Y, c.Zoom = 10, 20, 2
	wx, wy := c.ScreenToWorld(30, 40)
	if math.Abs(wx-25) > eps || math.Abs(wy-40) > eps {
		t.Fatalf("world = %v %v", wx, wy)
	}
	g := c.GeoM()
	sx, sy := g.Apply(wx, wy)
	if math.Abs(sx-30) > eps || math.Abs(sy-40) > eps {
		t.Fatalf("screen = %v %v", sx, sy)
	}
}

func TestZoomAtAnchorsCursor(t *testing.T) {
	c := camera.New()
	const worldW, worldH = 100.0, 100.0
	const sx, sy = 40.0, 30.0
	wx, wy := c.ScreenToWorld(sx, sy)
	c.ZoomAt(2, sx, sy, worldW, worldH)
	if c.Zoom != 2 {
		t.Fatalf("Zoom = %v", c.Zoom)
	}
	nwx, nwy := c.ScreenToWorld(sx, sy)
	if math.Abs(nwx-wx) > eps || math.Abs(nwy-wy) > eps {
		t.Fatalf("anchor drifted: %v,%v -> %v,%v", wx, wy, nwx, nwy)
	}
}

func TestZoomClamped(t *testing.T) {
	c := camera.New()
	c.ZoomAt(0.1, 0, 0, 100, 100)
	if c.Zoom != c.MinZoom {
		t.Fatalf("Zoom = %v", c.Zoom)
	}
	for range 10 {
		c.ZoomAt(3, 0, 0, 100, 100)
	}
	if c.Zoom != c.MaxZoom {
		t.Fatalf("Zoom = %v", c.Zoom)
	}
}

func TestPanClampedToWorld(t *testing.T) {
	c := camera.New()
	c.ZoomAt(2, 0, 0, 100, 100) // at zoom 2 the view covers 50 world units
	c.Pan(1e6, 1e6, 100, 100)
	if c.X != 50 || c.Y != 50 {
		t.Fatalf("pos = %v %v", c.X, c.Y)
	}
	c.Pan(-1e6, -1e6, 100, 100)
	if c.X != 0 || c.Y != 0 {
		t.Fatalf("pos = %v %v", c.X, c.Y)
	}
}

func TestNoZoomCannotPan(t *testing.T) {
	c := camera.New()
	c.Pan(500, 500, 100, 100)
	if c.X != 0 || c.Y != 0 {
		t.Fatalf("unzoomed camera must stay pinned: %v %v", c.X, c.Y)
	}
}
