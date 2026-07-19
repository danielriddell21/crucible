package raycast_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/raycast"
)

const eps = 1e-9

func TestNewCameraGeometry(t *testing.T) {
	fov := math.Pi / 2
	c := raycast.NewCamera(geom.Vec2{X: 5, Y: 5}, 0, fov)
	if math.Abs(c.DirX-1) > eps || math.Abs(c.DirY) > eps {
		t.Fatalf("dir = %v %v", c.DirX, c.DirY)
	}
	// The plane is perpendicular to the direction, with |plane| = tan(fov/2).
	if dot := c.DirX*c.PlaneX + c.DirY*c.PlaneY; math.Abs(dot) > eps {
		t.Fatalf("plane not perpendicular: dot = %v", dot)
	}
	planeLen := math.Hypot(c.PlaneX, c.PlaneY)
	if math.Abs(planeLen-math.Tan(fov/2)) > eps {
		t.Fatalf("plane length = %v", planeLen)
	}
}

func TestRayDirSweepsPlane(t *testing.T) {
	c := raycast.NewCamera(geom.Vec2{}, 0, math.Pi/2)
	lx, ly := c.RayDir(0, 100)
	cx, cy := c.RayDir(50, 100)
	rx, ry := c.RayDir(100, 100)
	if math.Abs(cx-c.DirX) > eps || math.Abs(cy-c.DirY) > eps {
		t.Fatalf("centre ray = %v %v", cx, cy)
	}
	// Facing +X, the left edge ray leans -Y and the right edge +Y.
	if !(ly < 0 && ry > 0 && lx > 0 && rx > 0) {
		t.Fatalf("edge rays = (%v,%v) (%v,%v)", lx, ly, rx, ry)
	}
}

// box is a 10x10 grid whose border cells are solid.
func box(x, y int) bool {
	return x <= 0 || y <= 0 || x >= 9 || y >= 9
}

func TestCastHitsFacingWall(t *testing.T) {
	pos := geom.Vec2{X: 5.5, Y: 5.5}
	h := raycast.Cast(pos, 1, 0, box) // straight +X into the east wall
	if h.Cell != (geom.Coord{X: 9, Y: 5}) {
		t.Fatalf("cell = %v", h.Cell)
	}
	if h.Prev != (geom.Coord{X: 8, Y: 5}) {
		t.Fatalf("prev = %v", h.Prev)
	}
	if math.Abs(h.Dist-3.5) > eps {
		t.Fatalf("dist = %v", h.Dist)
	}
	if h.Side != 0 {
		t.Fatalf("side = %d", h.Side)
	}
	if math.Abs(h.WallX-0.5) > eps {
		t.Fatalf("wallX = %v", h.WallX)
	}
}

func TestCastVerticalSide(t *testing.T) {
	pos := geom.Vec2{X: 5.25, Y: 5.5}
	h := raycast.Cast(pos, 0, 1, box) // straight +Y into the south wall
	if h.Cell != (geom.Coord{X: 5, Y: 9}) || h.Side != 1 {
		t.Fatalf("cell = %v side = %d", h.Cell, h.Side)
	}
	if math.Abs(h.Dist-3.5) > eps {
		t.Fatalf("dist = %v", h.Dist)
	}
	if math.Abs(h.WallX-0.25) > eps {
		t.Fatalf("wallX = %v", h.WallX)
	}
}

func TestCastPerpendicularDistance(t *testing.T) {
	// A diagonal ray's Dist is the perpendicular distance, not the Euclidean
	// travel: for a 45° ray from the cell centre to a wall 3.5 cells away in
	// X, Dist is 3.5.
	pos := geom.Vec2{X: 5.5, Y: 5.5}
	h := raycast.Cast(pos, 1, 0.2, box)
	if h.Dist <= 3 || h.Dist > 4.5 {
		t.Fatalf("dist = %v", h.Dist)
	}
}

func TestCastEscapesMalformedMap(t *testing.T) {
	open := func(x, y int) bool { return false }
	h := raycast.Cast(geom.Vec2{X: 0.5, Y: 0.5}, 1, 0, open)
	if h.Dist <= 0 {
		t.Fatalf("dist = %v", h.Dist)
	}
}

func TestProjectAheadAndBehind(t *testing.T) {
	c := raycast.NewCamera(geom.Vec2{X: 0, Y: 0}, 0, math.Pi/2)
	p, ok := c.Project(geom.Vec2{X: 4, Y: 0}, 320, 200, 1)
	if !ok {
		t.Fatal("billboard dead ahead must project")
	}
	if p.ScreenX != 160 {
		t.Fatalf("ScreenX = %d", p.ScreenX)
	}
	if math.Abs(p.Depth-4) > eps {
		t.Fatalf("Depth = %v", p.Depth)
	}
	if p.Size != 50 { // h / depth * scale = 200/4
		t.Fatalf("Size = %d", p.Size)
	}
	if _, ok := c.Project(geom.Vec2{X: -4, Y: 0}, 320, 200, 1); ok {
		t.Fatal("billboard behind the camera must not project")
	}
}

func TestProjectOffCentre(t *testing.T) {
	c := raycast.NewCamera(geom.Vec2{X: 0, Y: 0}, 0, math.Pi/2)
	right, ok := c.Project(geom.Vec2{X: 4, Y: 1}, 320, 200, 1)
	if !ok {
		t.Fatal("must project")
	}
	if right.ScreenX <= 160 {
		t.Fatalf("billboard to the camera's right must land right of centre: %d", right.ScreenX)
	}
}

func TestProjectTooSmall(t *testing.T) {
	c := raycast.NewCamera(geom.Vec2{}, 0, math.Pi/2)
	if _, ok := c.Project(geom.Vec2{X: 150, Y: 0}, 320, 200, 1); ok {
		t.Fatal("a distant speck must be rejected")
	}
}

func TestSortFarToNear(t *testing.T) {
	items := []float64{1, 4, 2, 9, 3}
	raycast.SortFarToNear(items, func(v float64) float64 { return v })
	want := []float64{9, 4, 3, 2, 1}
	for i, v := range items {
		if v != want[i] {
			t.Fatalf("items = %v", items)
		}
	}
}
