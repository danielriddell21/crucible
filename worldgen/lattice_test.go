package worldgen_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

func cfg(thin float64) worldgen.LatticeConfig {
	return worldgen.LatticeConfig{Cols: 9, Rows: 9, MinSpan: 80, MaxSpan: 150, Thin: thin}
}

func newLattice(t *testing.T, seed uint64, cols, rows int) *worldgen.Lattice {
	t.Helper()
	c := cfg(0)
	c.Cols, c.Rows = cols, rows
	return worldgen.NewLattice(seed, c)
}

func TestLatticeIsComplete(t *testing.T) {
	const cols, rows = 5, 4
	l := newLattice(t, 1, cols, rows)

	if got := len(l.Nodes); got != cols*rows {
		t.Errorf("nodes = %d, want %d", got, cols*rows)
	}
	if want := (cols-1)*rows + cols*(rows-1); len(l.Edges) != want {
		t.Errorf("edges = %d, want %d", len(l.Edges), want)
	}
}

func TestLatticeCornerAndInteriorDegrees(t *testing.T) {
	l := newLattice(t, 2, 4, 4)
	byGrid := map[[2]int]*worldgen.LatticeNode{}
	for _, n := range l.Nodes {
		byGrid[[2]int{n.Col, n.Row}] = n
	}
	corner := byGrid[[2]int{-2, -2}]
	if corner == nil || corner.Degree() != 2 {
		t.Errorf("corner degree = %v, want 2", corner)
	}
	if got := byGrid[[2]int{-1, -1}].Degree(); got != 4 {
		t.Errorf("interior degree = %d, want 4", got)
	}
}

func TestSpansVaryAndStayInRange(t *testing.T) {
	l := newLattice(t, 4, 12, 3)
	xs := map[int]float64{}
	for _, n := range l.Nodes {
		xs[n.Col] = n.Pos.X
	}
	first, varied := math.NaN(), false
	for i := -6; i+1 < 6; i++ {
		gap := xs[i+1] - xs[i]
		if gap < 80-1e-9 || gap > 150+1e-9 {
			t.Errorf("gap %v is outside [80, 150]", gap)
		}
		if math.IsNaN(first) {
			first = gap
		} else if math.Abs(gap-first) > 1e-9 {
			varied = true
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
		if got := a.Pos.Dist(b.Pos); math.Abs(got-e.Length) > 1e-9 {
			t.Errorf("edge %d Length = %v, want %v", e.ID, e.Length, got)
		}
		if math.Abs(e.Dir.Len()-1) > 1e-9 {
			t.Errorf("edge %d Dir is not a unit vector: %v", e.ID, e.Dir)
		}
		switch e.Axis {
		case worldgen.AxisX:
			if a.Row != b.Row || b.Col != a.Col+1 || e.Dir.X <= 0 {
				t.Errorf("AxisX edge %d is wrong: %v to %v", e.ID, a, b)
			}
		case worldgen.AxisY:
			if a.Col != b.Col || b.Row != a.Row+1 || e.Dir.Y <= 0 {
				t.Errorf("AxisY edge %d is wrong: %v to %v", e.ID, a, b)
			}
		}
	}
}

// TestGrowthAgreesWithWholeCloth is the property the whole design exists for:
// a lattice grown a patch at a time must be identical to one generated in a
// single pass, or a world that extends as you drive would not line up with
// itself.
func TestGrowthAgreesWithWholeCloth(t *testing.T) {
	for _, thin := range []float64{0, 0.15} {
		whole := worldgen.NewGrowable(11, cfg(thin))
		whole.Grow(geom.Rect{X: -4, Y: -4, W: 9, H: 9})

		// The same region, reached in four overlapping pieces in an order no
		// single-pass generation would ever use.
		piece := worldgen.NewGrowable(11, cfg(thin))
		for _, r := range []geom.Rect{
			{X: 0, Y: 0, W: 5, H: 5},
			{X: -4, Y: -4, W: 5, H: 5},
			{X: -4, Y: 0, W: 5, H: 5},
			{X: 0, Y: -4, W: 5, H: 5},
		} {
			piece.Grow(r)
		}

		if len(whole.Nodes) != len(piece.Nodes) {
			t.Fatalf("thin=%v: %d nodes grown, %d whole", thin, len(piece.Nodes), len(whole.Nodes))
		}
		if len(whole.Edges) != len(piece.Edges) {
			t.Fatalf("thin=%v: %d edges grown, %d whole", thin, len(piece.Edges), len(whole.Edges))
		}

		// Compare by cell rather than by ID: the pieces were reached in a
		// different order, so the numbering differs even though the graph
		// does not.
		wholeEdges := edgeSet(whole)
		pieceEdges := edgeSet(piece)
		for k := range wholeEdges {
			if !pieceEdges[k] {
				t.Errorf("thin=%v: edge %v missing from the grown lattice", thin, k)
			}
		}
		for k := range pieceEdges {
			if !wholeEdges[k] {
				t.Errorf("thin=%v: grown lattice invented edge %v", thin, k)
			}
		}
		for _, n := range piece.Nodes {
			w := whole.Node(n.Col, n.Row)
			if w == nil || w.Pos != n.Pos {
				t.Errorf("thin=%v: node (%d,%d) at %v, want %v", thin, n.Col, n.Row, n.Pos, w)
			}
		}
	}
}

// edgeSet keys every edge by the cells it joins, which is independent of the
// order the lattice was built in.
func edgeSet(l *worldgen.Lattice) map[[4]int]bool {
	out := map[[4]int]bool{}
	for _, e := range l.Edges {
		a, b := l.Nodes[e.A], l.Nodes[e.B]
		out[[4]int{a.Col, a.Row, b.Col, b.Row}] = true
	}
	return out
}

func TestGrowIsIdempotent(t *testing.T) {
	l := worldgen.NewGrowable(12, cfg(0.15))
	r := geom.Rect{X: 0, Y: 0, W: 6, H: 6}
	l.Grow(r)
	nodes, edges := len(l.Nodes), len(l.Edges)

	if added := l.Grow(r); len(added) != 0 {
		t.Errorf("regrowing the same cells added %d nodes", len(added))
	}
	if len(l.Nodes) != nodes || len(l.Edges) != edges {
		t.Errorf("regrowing changed the lattice: %d/%d, was %d/%d",
			len(l.Nodes), len(l.Edges), nodes, edges)
	}
}

func TestGrowJoinsAcrossTheSeam(t *testing.T) {
	// Two touching regions must end up connected, or a world would grow in
	// islands a driver could never reach.
	l := worldgen.NewGrowable(13, cfg(0))
	l.Grow(geom.Rect{X: 0, Y: 0, W: 3, H: 3})
	l.Grow(geom.Rect{X: 3, Y: 0, W: 3, H: 3})

	joined := false
	for _, e := range l.Edges {
		a, b := l.Nodes[e.A], l.Nodes[e.B]
		if a.Col == 2 && b.Col == 3 && a.Row == b.Row {
			joined = true
		}
	}
	if !joined {
		t.Error("the two regions were never joined across the seam")
	}
}

func TestIDsSurviveGrowth(t *testing.T) {
	// Anything holding an ID — an agent on a road, a route across town — must
	// still hold the same thing after the world extends.
	l := worldgen.NewGrowable(14, cfg(0.15))
	l.Grow(geom.Rect{X: 0, Y: 0, W: 4, H: 4})

	before := make([]worldgen.LatticeEdge, len(l.Edges))
	for i, e := range l.Edges {
		before[i] = *e
	}
	nodesBefore := make([]geom.Vec2, len(l.Nodes))
	for i, n := range l.Nodes {
		nodesBefore[i] = n.Pos
	}

	l.Grow(geom.Rect{X: -6, Y: -6, W: 14, H: 14})

	for i, was := range before {
		now := l.Edges[i]
		if now.ID != was.ID || now.A != was.A || now.B != was.B || now.Axis != was.Axis {
			t.Fatalf("edge %d changed identity: %+v, was %+v", i, *now, was)
		}
	}
	for i, was := range nodesBefore {
		if l.Nodes[i].Pos != was {
			t.Fatalf("node %d moved from %v to %v", i, was, l.Nodes[i].Pos)
		}
	}
}

func TestThinningNeverStrandsANode(t *testing.T) {
	// Every node in an infinite grid starts with four edges. No node may lose
	// two, so nothing drops below three and there are no dead ends.
	l := worldgen.NewGrowable(15, cfg(0.4))
	l.Grow(geom.Rect{X: -8, Y: -8, W: 17, H: 17})

	for _, n := range l.Nodes {
		// Only judge the interior: nodes on the edge of what has been grown
		// are missing neighbours that simply do not exist yet.
		if n.Col <= -8 || n.Col >= 8 || n.Row <= -8 || n.Row >= 8 {
			continue
		}
		if n.Degree() < 3 {
			t.Errorf("interior node (%d,%d) left with degree %d", n.Col, n.Row, n.Degree())
		}
	}
}

func TestThinningActuallyRemovesEdges(t *testing.T) {
	full := worldgen.NewGrowable(16, cfg(0))
	full.Grow(geom.Rect{X: -8, Y: -8, W: 17, H: 17})
	thinned := worldgen.NewGrowable(16, cfg(0.3))
	thinned.Grow(geom.Rect{X: -8, Y: -8, W: 17, H: 17})

	if len(thinned.Edges) >= len(full.Edges) {
		t.Fatalf("thinning removed nothing: %d edges, full has %d",
			len(thinned.Edges), len(full.Edges))
	}
	// A rule that guarantees at most one removal per node cannot reach the
	// full share, but it should get somewhere near a quarter of the way.
	dropped := float64(len(full.Edges)-len(thinned.Edges)) / float64(len(full.Edges))
	if dropped < 0.05 {
		t.Errorf("only %.1f%% of edges went, want a visible share", dropped*100)
	}
	t.Logf("thin 0.3 dropped %.1f%% of edges", dropped*100)
}

func TestThinZeroKeepsEverything(t *testing.T) {
	l := worldgen.NewGrowable(17, cfg(0))
	l.Grow(geom.Rect{X: 0, Y: 0, W: 5, H: 5})
	if want := 2 * 5 * 4; len(l.Edges) != want {
		t.Errorf("edges = %d, want %d", len(l.Edges), want)
	}
}

func TestCellRoundTrip(t *testing.T) {
	l := worldgen.NewGrowable(18, cfg(0))
	l.Grow(geom.Rect{X: -5, Y: -5, W: 11, H: 11})
	for _, n := range l.Nodes {
		col, row := l.Cell(n.Pos)
		if col != n.Col || row != n.Row {
			t.Errorf("Cell(%v) = (%d,%d), want (%d,%d)", n.Pos, col, row, n.Col, n.Row)
		}
	}
}

func TestCellsAroundCoversTheRadius(t *testing.T) {
	l := worldgen.NewGrowable(19, cfg(0))
	at := geom.Vec2{X: 500, Y: -300}
	const radius = 400

	l.Grow(l.CellsAround(at, radius))
	// Every corner of the requested area has a node at or beyond it, so
	// nothing inside the radius is missing.
	if l.Min.X > at.X-radius || l.Min.Y > at.Y-radius {
		t.Errorf("grown area starts at %v, want it to cover %v", l.Min, at)
	}
	if l.Max.X < at.X+radius || l.Max.Y < at.Y+radius {
		t.Errorf("grown area ends at %v, want it to cover %v", l.Max, at)
	}
}

func TestGrowthIsDeterministic(t *testing.T) {
	a := worldgen.NewGrowable(20, cfg(0.2))
	b := worldgen.NewGrowable(20, cfg(0.2))
	r := geom.Rect{X: -3, Y: -3, W: 7, H: 7}
	a.Grow(r)
	b.Grow(r)

	if len(a.Edges) != len(b.Edges) {
		t.Fatalf("%d edges vs %d", len(a.Edges), len(b.Edges))
	}
	for i := range a.Edges {
		if *a.Edges[i] != *b.Edges[i] {
			t.Fatalf("edge %d differs: %+v vs %+v", i, *a.Edges[i], *b.Edges[i])
		}
	}
}

func TestFarFromTheOriginIsStillWellFormed(t *testing.T) {
	// A world that grows for an hour ends up a long way out; the arithmetic
	// has to keep working there.
	l := worldgen.NewGrowable(21, cfg(0.2))
	l.Grow(geom.Rect{X: 4000, Y: -9000, W: 6, H: 6})

	if len(l.Nodes) != 36 {
		t.Fatalf("nodes = %d, want 36", len(l.Nodes))
	}
	for _, e := range l.Edges {
		if e.Length < 80 || e.Length > 150 {
			t.Errorf("edge %d is %v long, outside the span range", e.ID, e.Length)
		}
	}
	for _, n := range l.Nodes {
		col, row := l.Cell(n.Pos)
		if col != n.Col || row != n.Row {
			t.Errorf("Cell round trip failed far out: (%d,%d) became (%d,%d)",
				n.Col, n.Row, col, row)
		}
	}
}

func TestOther(t *testing.T) {
	l := newLattice(t, 22, 3, 3)
	e := l.Edges[0]
	if got := l.Other(e, e.A); got != e.B {
		t.Errorf("Other(A) = %d, want %d", got, e.B)
	}
	if got := l.Other(e, e.B); got != e.A {
		t.Errorf("Other(B) = %d, want %d", got, e.A)
	}
}

func TestNodeLookup(t *testing.T) {
	l := worldgen.NewGrowable(23, cfg(0))
	l.Grow(geom.Rect{X: 0, Y: 0, W: 2, H: 2})
	if n := l.Node(1, 1); n == nil || n.Col != 1 || n.Row != 1 {
		t.Errorf("Node(1,1) = %v", n)
	}
	if n := l.Node(9, 9); n != nil {
		t.Errorf("Node(9,9) = %v, want nil for a cell never grown", n)
	}
}

func TestDegenerateConfig(t *testing.T) {
	l := worldgen.NewLattice(1, worldgen.LatticeConfig{
		Cols: 0, Rows: -3, MinSpan: 100, MaxSpan: 10,
	})
	if len(l.Nodes) != 4 || len(l.Edges) != 4 {
		t.Fatalf("got %d nodes and %d edges, want the smallest lattice", len(l.Nodes), len(l.Edges))
	}
}
