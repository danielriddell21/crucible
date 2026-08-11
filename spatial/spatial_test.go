package spatial_test

import (
	"math"
	"slices"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/spatial"
)

// collect gathers everything Near visits, sorted so comparisons do not depend
// on cell traversal order.
func collect(g *spatial.Grid[int], p geom.Vec2, radius float64) []int {
	var got []int
	g.Near(p, radius, func(v int) { got = append(got, v) })
	slices.Sort(got)
	return got
}

func TestNearFindsOnlyNearbyCells(t *testing.T) {
	g := spatial.New[int](10)
	g.Insert(geom.Vec2{X: 5, Y: 5}, 1)   // same cell as the query
	g.Insert(geom.Vec2{X: 15, Y: 5}, 2)  // the cell next door
	g.Insert(geom.Vec2{X: 500, Y: 5}, 3) // far away

	if got, want := collect(g, geom.Vec2{X: 5, Y: 5}, 6), []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("Near = %v, want %v", got, want)
	}
}

func TestNearIsABroadPhase(t *testing.T) {
	// A value in a touched cell is visited even when it lies beyond the
	// radius: the grid narrows the search, the caller does the exact test.
	g := spatial.New[int](10)
	g.Insert(geom.Vec2{X: 9, Y: 9}, 1)

	origin := geom.Vec2{}
	if got := collect(g, origin, 1); !slices.Equal(got, []int{1}) {
		t.Fatalf("Near = %v, want the whole cell's contents", got)
	}
	if d := origin.Dist(geom.Vec2{X: 9, Y: 9}); d <= 1 {
		t.Fatalf("test is not exercising the broad phase: distance %v", d)
	}
}

func TestNearZeroRadiusVisitsTheOwnCell(t *testing.T) {
	g := spatial.New[int](10)
	g.Insert(geom.Vec2{X: 3, Y: 3}, 1)
	if got := collect(g, geom.Vec2{X: 7, Y: 7}, 0); !slices.Equal(got, []int{1}) {
		t.Errorf("Near = %v, want [1]", got)
	}
}

func TestCellsAreUniformAcrossTheOrigin(t *testing.T) {
	// Truncation toward zero would make [-10,0) and [0,10) share a cell,
	// doubling its width. Two points 1 apart across the origin must land in
	// different cells of a grid this fine.
	g := spatial.New[int](0.5)
	g.Insert(geom.Vec2{X: -0.75, Y: 0}, 1)
	g.Insert(geom.Vec2{X: 0.75, Y: 0}, 2)

	if got := collect(g, geom.Vec2{X: -0.75, Y: 0}, 0); !slices.Equal(got, []int{1}) {
		t.Errorf("negative side = %v, want [1] alone", got)
	}
	if got := collect(g, geom.Vec2{X: 0.75, Y: 0}, 0); !slices.Equal(got, []int{2}) {
		t.Errorf("positive side = %v, want [2] alone", got)
	}
}

func TestClearKeepsCapacity(t *testing.T) {
	g := spatial.New[int](10)
	for i := range 32 {
		g.Insert(geom.Vec2{X: 1, Y: 1}, i)
	}
	if g.Len() != 32 {
		t.Fatalf("Len = %d, want 32", g.Len())
	}

	g.Clear()
	if g.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", g.Len())
	}
	if got := collect(g, geom.Vec2{X: 1, Y: 1}, 0); got != nil {
		t.Errorf("Near after Clear = %v, want nothing", got)
	}

	// Refilling to the previous size must not allocate, which is the point of
	// keeping the cells rather than dropping them.
	allocs := testing.AllocsPerRun(50, func() {
		g.Clear()
		for i := range 32 {
			g.Insert(geom.Vec2{X: 1, Y: 1}, i)
		}
	})
	if allocs != 0 {
		t.Errorf("refill allocated %v times, want 0", allocs)
	}
}

func TestTorusWrapsAtTheEdges(t *testing.T) {
	// A 100x100 world in 10-unit cells. A value just past the right edge is a
	// short hop from a query just inside the left one.
	g := spatial.NewTorus[int](100, 100, 10)
	g.Insert(geom.Vec2{X: 95, Y: 50}, 1)

	if got := collect(g, geom.Vec2{X: 5, Y: 50}, 12); !slices.Equal(got, []int{1}) {
		t.Errorf("wrapped query = %v, want [1]", got)
	}
}

func TestTorusFoldsOutOfBoundsPositions(t *testing.T) {
	g := spatial.NewTorus[int](100, 100, 10)
	g.Insert(geom.Vec2{X: 105, Y: -5}, 1) // folds to (5, 95)

	if got := collect(g, geom.Vec2{X: 5, Y: 95}, 0); !slices.Equal(got, []int{1}) {
		t.Errorf("Near = %v, want [1] at the folded position", got)
	}
}

func TestTorusWideQueryVisitsEachCellOnce(t *testing.T) {
	// A radius wider than the world laps it. Capping the span keeps each
	// value visited once rather than once per lap.
	g := spatial.NewTorus[int](100, 100, 10)
	g.Insert(geom.Vec2{X: 50, Y: 50}, 1)

	if got := collect(g, geom.Vec2{X: 50, Y: 50}, 1000); !slices.Equal(got, []int{1}) {
		t.Errorf("Near = %v, want [1] exactly once", got)
	}
}

func TestTorusClearReusesCells(t *testing.T) {
	g := spatial.NewTorus[int](100, 100, 10)
	g.Insert(geom.Vec2{X: 5, Y: 5}, 1)
	g.Clear()
	if g.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", g.Len())
	}
	if got := collect(g, geom.Vec2{X: 5, Y: 5}, 0); got != nil {
		t.Errorf("Near after Clear = %v, want nothing", got)
	}
}

func TestInsertOnceSkipsConsecutiveRepeats(t *testing.T) {
	// Walking a shape and inserting at each step: the repeats within one cell
	// collapse, but the value is still filed in every cell it crosses.
	g := spatial.New[int](10)
	for s := 0.0; s <= 30; s += 1 {
		spatial.InsertOnce(g, geom.Vec2{X: s, Y: 0}, 7)
	}
	if g.Len() != 4 { // cells 0, 1, 2 and 3
		t.Errorf("Len = %d, want one entry per crossed cell (4)", g.Len())
	}
	for _, x := range []float64{5, 15, 25} {
		if got := collect(g, geom.Vec2{X: x, Y: 0}, 0); !slices.Equal(got, []int{7}) {
			t.Errorf("cell at x=%v = %v, want [7]", x, got)
		}
	}
}

func TestInsertOnceKeepsDistinctValues(t *testing.T) {
	// Only a repeat of the most recent value is skipped; two different values
	// in one cell both stay.
	g := spatial.New[int](10)
	spatial.InsertOnce(g, geom.Vec2{X: 1, Y: 1}, 1)
	spatial.InsertOnce(g, geom.Vec2{X: 2, Y: 2}, 2)
	spatial.InsertOnce(g, geom.Vec2{X: 3, Y: 3}, 1)

	if got := collect(g, geom.Vec2{X: 1, Y: 1}, 0); !slices.Equal(got, []int{1, 1, 2}) {
		t.Errorf("Near = %v, want [1 1 2]", got)
	}
}

func TestInsertOnceOnATorus(t *testing.T) {
	g := spatial.NewTorus[int](100, 100, 10)
	for range 5 {
		spatial.InsertOnce(g, geom.Vec2{X: 5, Y: 5}, 3)
	}
	if g.Len() != 1 {
		t.Errorf("Len = %d, want 1", g.Len())
	}
}

func TestNonPositiveCellSizeIsUsable(t *testing.T) {
	// A zero cell size would divide by zero on every insert; the grid falls
	// back to a cell of one rather than producing infinities.
	for _, cell := range []float64{0, -5} {
		g := spatial.New[int](cell)
		if g.Cell() != 1 {
			t.Errorf("New(%v).Cell() = %v, want 1", cell, g.Cell())
		}
		g.Insert(geom.Vec2{X: 0.5, Y: 0.5}, 1)
		if got := collect(g, geom.Vec2{X: 0.5, Y: 0.5}, 0); !slices.Equal(got, []int{1}) {
			t.Errorf("New(%v) lost the value: %v", cell, got)
		}
	}
}

func TestNegativeRadiusIsTreatedAsZero(t *testing.T) {
	g := spatial.New[int](10)
	g.Insert(geom.Vec2{X: 5, Y: 5}, 1)
	if got := collect(g, geom.Vec2{X: 5, Y: 5}, -3); !slices.Equal(got, []int{1}) {
		t.Errorf("Near = %v, want the own cell", got)
	}
}

func TestGridNarrowsALargeWorld(t *testing.T) {
	// The reason the package exists: a query over ten thousand points visits
	// a tiny fraction of them.
	g := spatial.New[int](10)
	const side = 100
	for y := range side {
		for x := range side {
			g.Insert(geom.Vec2{X: float64(x) * 10, Y: float64(y) * 10}, y*side+x)
		}
	}

	visited := 0
	g.Near(geom.Vec2{X: 500, Y: 500}, 15, func(int) { visited++ })
	if visited == 0 || visited > 25 {
		t.Errorf("visited %d of %d points, want a small neighbourhood", visited, side*side)
	}
}

func TestTorusMatchesToroidalDistance(t *testing.T) {
	// The grid's idea of "near" on a torus must agree with geom's: everything
	// within the radius by ToroidalDist is visited.
	const w, h = 100.0, 100.0
	g := spatial.NewTorus[int](w, h, 10)
	pts := []geom.Vec2{{X: 2, Y: 2}, {X: 98, Y: 98}, {X: 50, Y: 50}, {X: 99, Y: 3}}
	for i, p := range pts {
		g.Insert(p, i)
	}

	const radius = 8
	q := geom.Vec2{X: 1, Y: 1}
	seen := map[int]bool{}
	g.Near(q, radius, func(v int) { seen[v] = true })

	for i, p := range pts {
		if d := q.ToroidalDist(p, w, h); d <= radius && !seen[i] {
			t.Errorf("point %d at %v is %v away but was not visited", i, p, math.Round(d))
		}
	}
}

func TestRemove(t *testing.T) {
	g := spatial.New[int](10)
	at := geom.Vec2{X: 5, Y: 5}
	g.Insert(at, 1)
	g.Insert(at, 2)

	if !spatial.Remove(g, at, 1) {
		t.Fatal("Remove reported nothing to remove")
	}
	if g.Len() != 1 {
		t.Errorf("Len = %d, want 1", g.Len())
	}
	if got := collect(g, at, 0); !slices.Equal(got, []int{2}) {
		t.Errorf("Near = %v, want [2]", got)
	}
	// Removing what is not there changes nothing.
	if spatial.Remove(g, at, 99) {
		t.Error("Remove found a value that was never inserted")
	}
	if spatial.Remove(g, geom.Vec2{X: 500, Y: 500}, 2) {
		t.Error("Remove found a value in the wrong cell")
	}
	if g.Len() != 1 {
		t.Errorf("Len = %d after failed removals, want 1", g.Len())
	}
}

func TestRemoveOnATorus(t *testing.T) {
	g := spatial.NewTorus[int](100, 100, 10)
	at := geom.Vec2{X: 105, Y: -5} // folds to (5, 95)
	g.Insert(at, 7)

	if !spatial.Remove(g, geom.Vec2{X: 5, Y: 95}, 7) {
		t.Fatal("Remove did not fold the position the way Insert did")
	}
	if g.Len() != 0 {
		t.Errorf("Len = %d, want 0", g.Len())
	}
}

func TestRemoveKeepsTheRestInOrder(t *testing.T) {
	// InsertOnce compares against the last value in a cell, so removing from
	// the middle must not reshuffle what is left.
	g := spatial.New[int](10)
	at := geom.Vec2{X: 1, Y: 1}
	for _, v := range []int{1, 2, 3, 4} {
		g.Insert(at, v)
	}
	spatial.Remove(g, at, 2)

	var got []int
	g.Near(at, 0, func(v int) { got = append(got, v) })
	if !slices.Equal(got, []int{1, 3, 4}) {
		t.Errorf("order = %v, want [1 3 4]", got)
	}
}

func TestRemoveThenInsertOnce(t *testing.T) {
	g := spatial.New[int](10)
	at := geom.Vec2{X: 1, Y: 1}
	spatial.InsertOnce(g, at, 5)
	spatial.Remove(g, at, 5)
	spatial.InsertOnce(g, at, 5)
	if g.Len() != 1 {
		t.Errorf("Len = %d, want the value back after removal", g.Len())
	}
}
