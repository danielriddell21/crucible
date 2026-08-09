package worldgen_test

import (
	"testing"

	"github.com/danielriddell21/crucible/worldgen"
)

func newLattice(t *testing.T, seed int64, cols, rows int) *worldgen.Lattice {
	t.Helper()
	return worldgen.NewLattice(worldgen.NewRNG(seed), worldgen.LatticeConfig{
		Cols: cols, Rows: rows, MinSpan: 80, MaxSpan: 150,
	})
}

func TestLatticeIsComplete(t *testing.T) {
	const cols, rows = 5, 4
	l := newLattice(t, 1, cols, rows)

	if got := len(l.Nodes); got != cols*rows {
		t.Errorf("nodes = %d, want %d", got, cols*rows)
	}
	// Every horizontal and vertical neighbour pair is joined.
	want := (cols-1)*rows + cols*(rows-1)
	if got := len(l.Edges); got != want {
		t.Errorf("edges = %d, want %d", got, want)
	}
	if len(l.Xs) != cols || len(l.Ys) != rows {
		t.Errorf("grid lines = %dx%d, want %dx%d", len(l.Xs), len(l.Ys), cols, rows)
	}
}

func TestLatticeCornerAndInteriorDegrees(t *testing.T) {
	l := newLattice(t, 2, 4, 4)

	byGrid := map[[2]int]*worldgen.LatticeNode{}
	for _, n := range l.Nodes {
		byGrid[[2]int{n.Col, n.Row}] = n
	}
	if got := byGrid[[2]int{0, 0}].Degree(); got != 2 {
		t.Errorf("corner degree = %d, want 2", got)
	}
	if got := byGrid[[2]int{1, 0}].Degree(); got != 3 {
		t.Errorf("edge-of-grid degree = %d, want 3", got)
	}
	if got := byGrid[[2]int{1, 1}].Degree(); got != 4 {
		t.Errorf("interior degree = %d, want 4", got)
	}
}

func TestLatticeIsCentredOnTheOrigin(t *testing.T) {
	l := newLattice(t, 3, 7, 7)
	// The extremes are symmetric about zero, so a camera at the origin starts
	// in the middle of the world.
	if got := l.Min.X + l.Max.X; got > 1e-9 || got < -1e-9 {
		t.Errorf("X extremes sum to %v, want the lattice centred", got)
	}
	if got := l.Min.Y + l.Max.Y; got > 1e-9 || got < -1e-9 {
		t.Errorf("Y extremes sum to %v, want the lattice centred", got)
	}
}

func TestLatticeSpansVary(t *testing.T) {
	// Uniform spacing would read as graph paper; the gaps must differ.
	l := newLattice(t, 4, 9, 9)
	first := l.Xs[1] - l.Xs[0]
	varied := false
	for i := 1; i+1 < len(l.Xs); i++ {
		if gap := l.Xs[i+1] - l.Xs[i]; gap < first-1e-9 || gap > first+1e-9 {
			varied = true
		}
		if gap := l.Xs[i+1] - l.Xs[i]; gap < 80 || gap > 150 {
			t.Errorf("gap %v is outside [80, 150]", gap)
		}
	}
	if !varied {
		t.Error("every gap was identical, want varied spans")
	}
}

func TestLatticeEdgeGeometry(t *testing.T) {
	l := newLattice(t, 5, 4, 4)
	for _, e := range l.Edges {
		a, b := l.Nodes[e.A], l.Nodes[e.B]
		if got := a.Pos.Dist(b.Pos); got < e.Length-1e-9 || got > e.Length+1e-9 {
			t.Errorf("edge %d Length = %v, want %v", e.ID, e.Length, got)
		}
		if got := e.Dir.Len(); got < 1-1e-9 || got > 1+1e-9 {
			t.Errorf("edge %d Dir is not a unit vector: %v", e.ID, e.Dir)
		}
		switch e.Axis {
		case worldgen.AxisX:
			if a.Row != b.Row || b.Col != a.Col+1 {
				t.Errorf("AxisX edge joins (%d,%d) to (%d,%d)", a.Col, a.Row, b.Col, b.Row)
			}
			if e.Dir.X <= 0 {
				t.Errorf("AxisX edge %d points backwards: %v", e.ID, e.Dir)
			}
		case worldgen.AxisY:
			if a.Col != b.Col || b.Row != a.Row+1 {
				t.Errorf("AxisY edge joins (%d,%d) to (%d,%d)", a.Col, a.Row, b.Col, b.Row)
			}
			if e.Dir.Y <= 0 {
				t.Errorf("AxisY edge %d points backwards: %v", e.ID, e.Dir)
			}
		}
	}
}

func TestLatticeIsDeterministic(t *testing.T) {
	a := newLattice(t, 99, 6, 6)
	b := newLattice(t, 99, 6, 6)
	for i := range a.Nodes {
		if a.Nodes[i].Pos != b.Nodes[i].Pos {
			t.Fatalf("node %d differs: %v vs %v", i, a.Nodes[i].Pos, b.Nodes[i].Pos)
		}
	}
}

func TestThinRemovesEdgesWithoutStrandingNodes(t *testing.T) {
	l := newLattice(t, 7, 9, 9)
	before := len(l.Edges)

	got := l.Thin(worldgen.NewRNG(7), 0.15, 3, nil)
	if got == 0 {
		t.Fatal("Thin removed nothing")
	}
	if len(l.Edges) != before-got {
		t.Errorf("edges = %d after removing %d from %d", len(l.Edges), got, before)
	}
	for _, n := range l.Nodes {
		// A node that started below the floor keeps whatever it had; no
		// removal may take one below it.
		if n.Degree() < 2 {
			t.Errorf("node %d left with degree %d", n.ID, n.Degree())
		}
	}
}

func TestThinKeepsTheGraphConsistent(t *testing.T) {
	l := newLattice(t, 8, 8, 8)
	l.Thin(worldgen.NewRNG(8), 0.2, 3, nil)

	// Every edge ID a node holds must resolve, and resolve back to that node.
	for _, n := range l.Nodes {
		for _, id := range n.Edges {
			if id < 0 || id >= len(l.Edges) {
				t.Fatalf("node %d holds out-of-range edge %d", n.ID, id)
			}
			e := l.Edges[id]
			if e.ID != id {
				t.Errorf("edge at index %d reports ID %d", id, e.ID)
			}
			if e.A != n.ID && e.B != n.ID {
				t.Errorf("node %d holds edge %d, which joins %d and %d", n.ID, id, e.A, e.B)
			}
		}
	}
	// And every edge is held by both its endpoints.
	for _, e := range l.Edges {
		for _, end := range []int{e.A, e.B} {
			found := false
			for _, id := range l.Nodes[end].Edges {
				found = found || id == e.ID
			}
			if !found {
				t.Errorf("edge %d is not listed by its endpoint %d", e.ID, end)
			}
		}
	}
}

func TestThinRespectsTheRemovablePredicate(t *testing.T) {
	// The caller's own vocabulary decides what may go. Here: nothing on row
	// zero, standing in for "never remove a main road".
	l := newLattice(t, 9, 8, 8)
	protected := func(l *worldgen.Lattice, e *worldgen.LatticeEdge) bool {
		return l.Nodes[e.A].Row != 0 && l.Nodes[e.B].Row != 0
	}

	kept := map[[2]int]bool{}
	for _, e := range l.Edges {
		if !protected(l, e) {
			kept[[2]int{e.A, e.B}] = true
		}
	}
	l.Thin(worldgen.NewRNG(9), 0.5, 2, protected)

	still := map[[2]int]bool{}
	for _, e := range l.Edges {
		still[[2]int{e.A, e.B}] = true
	}
	for pair := range kept {
		if !still[pair] {
			t.Errorf("protected edge %v was removed", pair)
		}
	}
}

func TestThinHonoursMinDegree(t *testing.T) {
	// A high floor means nothing can go: every removal would take an endpoint
	// below it.
	l := newLattice(t, 10, 6, 6)
	before := len(l.Edges)
	if got := l.Thin(worldgen.NewRNG(10), 0.9, 4, nil); got != 0 {
		t.Errorf("Thin removed %d edges under a floor nothing can satisfy", got)
	}
	if len(l.Edges) != before {
		t.Errorf("edges = %d, want %d unchanged", len(l.Edges), before)
	}
}

func TestThinIsDeterministic(t *testing.T) {
	a, b := newLattice(t, 11, 7, 7), newLattice(t, 11, 7, 7)
	if a.Thin(worldgen.NewRNG(3), 0.2, 3, nil) != b.Thin(worldgen.NewRNG(3), 0.2, 3, nil) {
		t.Fatal("Thin removed different counts from identical lattices")
	}
	for i := range a.Edges {
		if a.Edges[i].A != b.Edges[i].A || a.Edges[i].B != b.Edges[i].B {
			t.Fatalf("edge %d differs after thinning", i)
		}
	}
}

func TestThinNothingToDo(t *testing.T) {
	l := newLattice(t, 12, 5, 5)
	before := len(l.Edges)
	for _, share := range []float64{0, -1} {
		if got := l.Thin(worldgen.NewRNG(1), share, 2, nil); got != 0 {
			t.Errorf("Thin(share=%v) removed %d", share, got)
		}
	}
	// A share too small to reach one whole edge also does nothing.
	if got := l.Thin(worldgen.NewRNG(1), 0.0001, 2, nil); got != 0 {
		t.Errorf("Thin of a fraction of an edge removed %d", got)
	}
	if len(l.Edges) != before {
		t.Errorf("edges = %d, want %d unchanged", len(l.Edges), before)
	}
}

func TestOther(t *testing.T) {
	l := newLattice(t, 13, 3, 3)
	e := l.Edges[0]
	if got := l.Other(e, e.A); got != e.B {
		t.Errorf("Other(A) = %d, want %d", got, e.B)
	}
	if got := l.Other(e, e.B); got != e.A {
		t.Errorf("Other(B) = %d, want %d", got, e.A)
	}
}

func TestLatticeDegenerateConfig(t *testing.T) {
	// Too few lines, and an inverted span range, must still produce a usable
	// lattice rather than panicking or returning nothing.
	l := worldgen.NewLattice(worldgen.NewRNG(1), worldgen.LatticeConfig{
		Cols: 0, Rows: -3, MinSpan: 100, MaxSpan: 10,
	})
	if len(l.Nodes) != 4 || len(l.Edges) != 4 {
		t.Fatalf("got %d nodes and %d edges, want the smallest lattice (4 and 4)",
			len(l.Nodes), len(l.Edges))
	}
	for i := 1; i < len(l.Xs); i++ {
		if l.Xs[i] <= l.Xs[i-1] {
			t.Errorf("grid lines are not increasing: %v", l.Xs)
		}
	}
}
