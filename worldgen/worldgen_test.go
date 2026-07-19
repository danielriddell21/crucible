package worldgen_test

import (
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

type grid struct {
	w, h int
	open []bool
}

func newGrid(w, h int) *grid { return &grid{w: w, h: h, open: make([]bool, w*h)} }

func (g *grid) Width() int  { return g.w }
func (g *grid) Height() int { return g.h }

func (g *grid) Open(x, y int) bool {
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return false
	}
	return g.open[y*g.w+x]
}

func (g *grid) Carve(x, y int) { g.open[y*g.w+x] = true }

func (g *grid) solid(c geom.Coord) bool { return !g.Open(c.X, c.Y) }

func generate(t *testing.T, seed int64) (*grid, []geom.Rect) {
	t.Helper()
	g := newGrid(48, 40)
	rooms := worldgen.Generate(worldgen.NewRNG(seed), g, worldgen.Config{})
	if len(rooms) < 2 {
		t.Fatalf("generated only %d rooms", len(rooms))
	}
	return g, rooms
}

func TestGenerateCarvesConnectedRooms(t *testing.T) {
	g, rooms := generate(t, 1)
	// Every room's centre must be open and reachable from the first room.
	src := rooms[0].Center()
	for i, r := range rooms {
		c := r.Center()
		if !g.Open(c.X, c.Y) {
			t.Fatalf("room %d centre %v not carved", i, c)
		}
		if !worldgen.Reachable(g.w, g.h, src, c, g.solid) {
			t.Fatalf("room %d centre %v unreachable from %v", i, c, src)
		}
	}
}

func TestGenerateRoomsInBounds(t *testing.T) {
	g, rooms := generate(t, 2)
	for _, r := range rooms {
		if r.X < 1 || r.Y < 1 || r.X+r.W > g.w-1 || r.Y+r.H > g.h-1 {
			t.Fatalf("room %+v touches the border", r)
		}
	}
}

func TestGenerateDeterministic(t *testing.T) {
	a, _ := generate(t, 7)
	b, _ := generate(t, 7)
	for i := range a.open {
		if a.open[i] != b.open[i] {
			t.Fatal("same seed produced different maps")
		}
	}
	c, _ := generate(t, 8)
	same := true
	for i := range a.open {
		if a.open[i] != c.open[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different seeds produced identical maps")
	}
}

func TestFloodDist(t *testing.T) {
	// 5x3 corridor with a wall in the middle column except one gap.
	g := newGrid(5, 3)
	for y := range 3 {
		for x := range 5 {
			g.Carve(x, y)
		}
	}
	wall := geom.Coord{X: 2, Y: 0}
	g.open[wall.Y*g.w+wall.X] = false
	g.open[1*g.w+2] = false // (2,1) also blocked; gap at (2,2)

	f := worldgen.FloodDist(g.w, g.h, geom.Coord{X: 0, Y: 0}, g.solid, nil)
	if got := f.At(geom.Coord{X: 1, Y: 0}); got != 1 {
		t.Fatalf("dist(1,0) = %d", got)
	}
	// Around the wall: (0,0)->(4,0) must detour through (2,2).
	if got := f.At(geom.Coord{X: 4, Y: 0}); got != 8 {
		t.Fatalf("dist(4,0) = %d", got)
	}
	if got := f.At(wall); got != -1 {
		t.Fatalf("dist(wall) = %d", got)
	}
	if got := f.At(geom.Coord{X: -1, Y: 0}); got != -1 {
		t.Fatalf("dist out of bounds = %d", got)
	}
}

func TestFloodDistBlockedSource(t *testing.T) {
	g := newGrid(3, 3)
	f := worldgen.FloodDist(g.w, g.h, geom.Coord{X: 1, Y: 1}, g.solid, nil)
	for _, d := range f.D {
		if d != -1 {
			t.Fatal("flood from a solid source must reach nothing")
		}
	}
}

func TestFloodDistStepGate(t *testing.T) {
	g := newGrid(3, 1)
	for x := range 3 {
		g.Carve(x, 0)
	}
	// A step rule that forbids entering the last column.
	step := func(from, to geom.Coord) bool { return to.X < 2 }
	f := worldgen.FloodDist(g.w, g.h, geom.Coord{X: 0, Y: 0}, g.solid, step)
	if got := f.At(geom.Coord{X: 1, Y: 0}); got != 1 {
		t.Fatalf("dist(1,0) = %d", got)
	}
	if got := f.At(geom.Coord{X: 2, Y: 0}); got != -1 {
		t.Fatalf("step-gated cell reached: %d", got)
	}
}

func TestStepsBetween(t *testing.T) {
	g := newGrid(4, 1)
	for x := range 4 {
		g.Carve(x, 0)
	}
	if got := worldgen.StepsBetween(g.w, g.h, geom.Coord{X: 0, Y: 0}, geom.Coord{X: 3, Y: 0}, g.solid); got != 3 {
		t.Fatalf("StepsBetween = %d", got)
	}
}

func TestRNGHelpers(t *testing.T) {
	r := worldgen.NewRNG(1)
	for range 100 {
		if v := r.Between(3, 5); v < 3 || v > 5 {
			t.Fatalf("Between out of range: %d", v)
		}
		if v := r.BetweenF(1, 2); v < 1 || v >= 2 {
			t.Fatalf("BetweenF out of range: %v", v)
		}
	}
	if r.Between(5, 3) != 5 {
		t.Error("Between with hi<=lo must return lo")
	}
	if r.BetweenF(5, 3) != 5 {
		t.Error("BetweenF with hi<=lo must return lo")
	}
	if r.Chance(0) || !r.Chance(1) {
		t.Error("Chance endpoints wrong")
	}
}
