package raycast_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/raycast"
)

// flatWorld is a 10x10 box room with optional per-cell overrides.
type flatWorld struct {
	floors map[geom.Coord]float64
	ceils  map[geom.Coord]float64
	tops   map[geom.Coord]float64
	extra  map[geom.Coord]bool // extra solid cells
}

func (w flatWorld) Floor(x, y int) float64 {
	if v, ok := w.floors[geom.Coord{X: x, Y: y}]; ok {
		return v
	}
	return 0
}

func (w flatWorld) Ceil(x, y int) float64 {
	if v, ok := w.ceils[geom.Coord{X: x, Y: y}]; ok {
		return v
	}
	return 1
}

func (w flatWorld) Solid(x, y int) bool {
	if w.extra[geom.Coord{X: x, Y: y}] {
		return true
	}
	return x <= 0 || y <= 0 || x >= 9 || y >= 9
}

func (w flatWorld) WallTop(x, y int) float64 { return w.tops[geom.Coord{X: x, Y: y}] }

// spanLog records every span a walk produced.
type spanLog struct {
	walls  []raycast.Face
	floors []geom.Coord
	ceils  []geom.Coord
	rows   map[int]int // screen row -> paint count, for coverage checks
}

func newSpanLog() *spanLog { return &spanLog{rows: map[int]int{}} }

func (s *spanLog) mark(y0, y1 int) {
	for y := y0; y <= y1; y++ {
		s.rows[y]++
	}
}

func (s *spanLog) WallSpan(x, y0, y1 int, face raycast.Face) {
	if y0 > y1 {
		panic("empty wall span delivered")
	}
	s.walls = append(s.walls, face)
	s.mark(y0, y1)
}

func (s *spanLog) FloorSpan(x, y0, y1 int, cell geom.Coord, z, dist float64) {
	s.floors = append(s.floors, cell)
	s.mark(y0, y1)
}

func (s *spanLog) CeilSpan(x, y0, y1 int, cell geom.Coord, z, dist float64) {
	s.ceils = append(s.ceils, cell)
	s.mark(y0, y1)
}

const (
	scrW, scrH = 320, 200
	eye        = 0.5
)

func centreCam() raycast.Camera {
	return raycast.NewCamera(geom.Vec2{X: 5.5, Y: 5.5}, 0, math.Pi/2)
}

func TestWalkColumnFlatRoom(t *testing.T) {
	log := newSpanLog()
	res := raycast.WalkColumn(centreCam(), scrW, scrH, scrW/2, eye, flatWorld{}, log)

	if math.Abs(res.Depth-3.5) > 1e-9 {
		t.Fatalf("Depth = %v", res.Depth)
	}
	if !math.IsInf(res.LowZ, 1) {
		t.Fatalf("LowZ = %v with no ledges", res.LowZ)
	}
	if len(log.walls) == 0 || len(log.floors) == 0 || len(log.ceils) == 0 {
		t.Fatalf("spans: %d walls %d floors %d ceils", len(log.walls), len(log.floors), len(log.ceils))
	}
	final := log.walls[len(log.walls)-1]
	if final.Cell != (geom.Coord{X: 9, Y: 5}) || final.Side != 0 {
		t.Fatalf("closing face = %+v", final)
	}
	// Every screen row is painted exactly once: floor, ceiling, and wall
	// spans tile the column with no gaps or overdraw.
	for y := range scrH {
		if log.rows[y] != 1 {
			t.Fatalf("row %d painted %d times", y, log.rows[y])
		}
	}
}

func TestWalkColumnStepFace(t *testing.T) {
	w := flatWorld{floors: map[geom.Coord]float64{
		{X: 7, Y: 5}: 0.25,
		{X: 8, Y: 5}: 0.25,
	}}
	log := newSpanLog()
	res := raycast.WalkColumn(centreCam(), scrW, scrH, scrW/2, eye, w, log)

	if math.Abs(res.Depth-3.5) > 1e-9 {
		t.Fatalf("Depth = %v", res.Depth)
	}
	if math.Abs(res.LowH-0.25) > 1e-9 || math.Abs(res.LowZ-1.5) > 1e-9 {
		t.Fatalf("ledge tracking = %+v", res)
	}
	// The rising step at the (6,5)->(7,5) boundary must have painted a face.
	found := false
	for _, f := range log.walls {
		if f.Cell == (geom.Coord{X: 7, Y: 5}) && math.Abs(f.Dist-1.5) < 1e-9 {
			found = true
		}
	}
	if !found {
		t.Fatal("no step face painted for the rise")
	}
	for y := range scrH {
		if log.rows[y] != 1 {
			t.Fatalf("row %d painted %d times", y, log.rows[y])
		}
	}
}

func TestWalkColumnSeesOverHalfWall(t *testing.T) {
	w := flatWorld{
		extra: map[geom.Coord]bool{{X: 7, Y: 5}: true},
		tops:  map[geom.Coord]float64{{X: 7, Y: 5}: 0.4},
	}
	log := newSpanLog()
	res := raycast.WalkColumn(centreCam(), scrW, scrH, scrW/2, eye, w, log)

	// The column must not close at the half wall (dist 1.5) but at the far
	// room wall (dist 3.5).
	if math.Abs(res.Depth-3.5) > 1e-9 {
		t.Fatalf("Depth = %v; column closed at the half wall", res.Depth)
	}
	if math.Abs(res.LowH-0.4) > 1e-9 || math.Abs(res.LowZ-1.5) > 1e-9 {
		t.Fatalf("half-wall occlusion = %+v", res)
	}
	// The wall's top surface reads as the floor of its own cell, and the
	// room beyond still shows its ceiling and far wall above the parapet
	// (its floor is correctly hidden behind the half wall).
	topPainted, beyondCeil := false, false
	for _, c := range log.floors {
		if c == (geom.Coord{X: 7, Y: 5}) {
			topPainted = true
		}
	}
	for _, c := range log.ceils {
		if c.X > 7 {
			beyondCeil = true
		}
	}
	if !topPainted {
		t.Fatal("half wall's top surface never painted")
	}
	if !beyondCeil {
		t.Fatal("ceiling beyond the half wall never painted")
	}
	if final := log.walls[len(log.walls)-1]; final.Cell != (geom.Coord{X: 9, Y: 5}) {
		t.Fatalf("closing face = %+v", final)
	}
}

func TestWalkColumnFullWallStopsRay(t *testing.T) {
	w := flatWorld{extra: map[geom.Coord]bool{{X: 7, Y: 5}: true}}
	log := newSpanLog()
	res := raycast.WalkColumn(centreCam(), scrW, scrH, scrW/2, eye, w, log)
	if math.Abs(res.Depth-1.5) > 1e-9 {
		t.Fatalf("Depth = %v", res.Depth)
	}
	for _, c := range log.floors {
		if c.X > 7 {
			t.Fatal("painted floor beyond a full wall")
		}
	}
}

func TestWalkColumnVerticalRaySafe(t *testing.T) {
	cam := raycast.NewCamera(geom.Vec2{X: 5.5, Y: 5.5}, math.Pi/2, math.Pi/2)
	log := newSpanLog()
	res := raycast.WalkColumn(cam, scrW, scrH, scrW/2, eye, flatWorld{}, log)
	if math.Abs(res.Depth-3.5) > 1e-9 {
		t.Fatalf("Depth = %v", res.Depth)
	}
	if f := log.walls[len(log.walls)-1]; f.Side != 1 {
		t.Fatalf("closing side = %d", f.Side)
	}
}

func TestWalkColumnEscapedRay(t *testing.T) {
	openHeights := heightsFunc{
		floor: func(x, y int) float64 { return 0 },
		ceil:  func(x, y int) float64 { return 1 },
		solid: func(x, y int) bool { return false },
	}
	log := newSpanLog()
	res := raycast.WalkColumn(centreCam(), scrW, scrH, scrW/2, eye, openHeights, log)
	if !math.IsInf(res.Depth, 1) {
		t.Fatalf("Depth = %v for an escaped ray", res.Depth)
	}
}

// heightsFunc adapts plain functions to the Heights interface.
type heightsFunc struct {
	floor func(x, y int) float64
	ceil  func(x, y int) float64
	solid func(x, y int) bool
}

func (h heightsFunc) Floor(x, y int) float64   { return h.floor(x, y) }
func (h heightsFunc) Ceil(x, y int) float64    { return h.ceil(x, y) }
func (h heightsFunc) Solid(x, y int) bool      { return h.solid(x, y) }
func (h heightsFunc) WallTop(x, y int) float64 { return 0 }
