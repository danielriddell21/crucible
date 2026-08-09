package worldgen

import "github.com/danielriddell21/crucible/geom"

// Axis names the direction a lattice edge runs in.
type Axis int

// The lattice axes.
const (
	// AxisX is an edge running along X, joining two nodes in the same row.
	AxisX Axis = iota
	// AxisY is an edge running along Y, joining two nodes in the same column.
	AxisY
)

// LatticeNode is one junction of a [Lattice].
type LatticeNode struct {
	// ID indexes the node in [Lattice.Nodes].
	ID int
	// Pos is the node's position in world units.
	Pos geom.Vec2
	// Col and Row are the node's place in the generating grid. Together with
	// [Lattice.Xs] and [Lattice.Ys] they are how a caller reasons about the
	// layout — "every third column is a main road", "the outer ring is
	// suburbs" — without measuring distances.
	Col, Row int
	// Edges holds the IDs of the edges meeting here.
	Edges []int
}

// Degree returns how many edges meet at the node.
func (n *LatticeNode) Degree() int { return len(n.Edges) }

// LatticeEdge joins two nodes of a [Lattice].
type LatticeEdge struct {
	// ID indexes the edge in [Lattice.Edges].
	ID int
	// A and B are the node IDs at each end. A is always the lower Col or Row,
	// so Dir points along increasing X or Y.
	A, B int
	// Axis is which way the edge runs.
	Axis Axis
	// Dir is the unit vector from A to B, and Length the distance between them.
	Dir    geom.Vec2
	Length float64
}

// Lattice is a seeded grid of nodes joined by axis-aligned edges, with the
// spacing between grid lines varied so the result reads as a place rather than
// as graph paper.
//
// It is the shared skeleton under a street plan, a district map, an overworld
// of connected regions: the engine owns the topology and the geometry, and the
// game attaches its own meaning to it. Nothing here knows what an edge is for.
// A driving game reads the edges as roads and gives each one lanes, a speed
// limit and a junction priority; a strategy map reads them as borders. The
// engine never learns either vocabulary.
//
// Build one with [NewLattice], optionally break up its regularity with
// [Lattice.Thin], then walk Nodes and Edges to build the world proper.
type Lattice struct {
	// Nodes and Edges are the graph. IDs index these slices.
	Nodes []*LatticeNode
	Edges []*LatticeEdge
	// Xs and Ys are the grid line positions, Cols and Rows of them. The cell
	// between consecutive lines is where a caller puts whatever fills the
	// space between edges — a city block, a field, a region.
	Xs, Ys []float64
	// Min and Max bound the node positions.
	Min, Max geom.Vec2
}

// LatticeConfig tunes [NewLattice]. Cols and Rows below two are raised to two,
// and a span range that is empty or inverted collapses to a uniform spacing.
type LatticeConfig struct {
	// Cols and Rows are how many grid lines to lay out on each axis, so the
	// full lattice has Cols×Rows nodes.
	Cols, Rows int
	// MinSpan and MaxSpan bound the gap between consecutive grid lines, in
	// world units. Each gap is drawn independently, which is what stops the
	// result looking like graph paper.
	MinSpan, MaxSpan float64
}

// NewLattice builds the complete grid: every node joined to its neighbours on
// both axes, with no edges missing. The lattice is centred on the origin, so a
// camera starting at (0, 0) starts in the middle of it.
//
// The same rng state always produces the same lattice.
func NewLattice(rng *RNG, cfg LatticeConfig) *Lattice {
	cols, rows := max(cfg.Cols, 2), max(cfg.Rows, 2)
	l := &Lattice{
		Xs: spans(rng, cols, cfg.MinSpan, cfg.MaxSpan),
		Ys: spans(rng, rows, cfg.MinSpan, cfg.MaxSpan),
	}
	centre(l.Xs)
	centre(l.Ys)

	ids := make([][]int, cols)
	for i := range cols {
		ids[i] = make([]int, rows)
		for j := range rows {
			n := &LatticeNode{ID: len(l.Nodes), Pos: geom.Vec2{X: l.Xs[i], Y: l.Ys[j]}, Col: i, Row: j}
			ids[i][j] = n.ID
			l.Nodes = append(l.Nodes, n)
		}
	}
	for i := range cols {
		for j := range rows {
			if i+1 < cols {
				l.addEdge(ids[i][j], ids[i+1][j], AxisX)
			}
			if j+1 < rows {
				l.addEdge(ids[i][j], ids[i][j+1], AxisY)
			}
		}
	}
	l.computeBounds()
	return l
}

// spans lays out n grid lines starting at zero, each a random gap after the
// last.
func spans(rng *RNG, n int, lo, hi float64) []float64 {
	out := make([]float64, n)
	var at float64
	for i := range n {
		out[i] = at
		at += rng.BetweenF(lo, hi)
	}
	return out
}

// centre shifts a run of grid lines so its midpoint sits on the origin.
func centre(lines []float64) {
	mid := lines[len(lines)-1] / 2
	for i := range lines {
		lines[i] -= mid
	}
}

func (l *Lattice) addEdge(a, b int, axis Axis) {
	na, nb := l.Nodes[a], l.Nodes[b]
	d := nb.Pos.Sub(na.Pos)
	e := &LatticeEdge{ID: len(l.Edges), A: a, B: b, Axis: axis, Dir: d.Normalize(), Length: d.Len()}
	l.Edges = append(l.Edges, e)
	na.Edges = append(na.Edges, e.ID)
	nb.Edges = append(nb.Edges, e.ID)
}

// Thin removes up to share of the edges at random, and returns how many it
// actually removed.
//
// An edge is only removed when both its endpoints would keep at least
// minDegree edges, so thinning never strands a node or leaves a dead end
// behind: the result is a less regular lattice, not a broken one. A minDegree
// of three keeps every surviving node a real junction; one merely keeps the
// graph free of isolated nodes.
//
// The optional removable predicate is where a caller's own vocabulary comes
// in — "only minor roads may go", "never remove a border" — and is consulted
// before the degree check. A nil predicate lets any edge be considered.
//
// Edge IDs are renumbered, because the surviving edges are compacted into a
// contiguous slice. Classify edges after thinning, not before, or hold on to
// endpoints rather than IDs.
func (l *Lattice) Thin(rng *RNG, share float64, minDegree int, removable func(*Lattice, *LatticeEdge) bool) int {
	if share <= 0 || len(l.Edges) == 0 {
		return 0
	}
	target := int(float64(len(l.Edges)) * min(share, 1))
	if target == 0 {
		return 0
	}

	removed := make(map[int]bool, target)
	for _, id := range perm(rng, len(l.Edges)) {
		if len(removed) >= target {
			break
		}
		e := l.Edges[id]
		if removable != nil && !removable(l, e) {
			continue
		}
		// The endpoints must survive the loss, so both need one to spare.
		if l.Nodes[e.A].Degree() <= minDegree || l.Nodes[e.B].Degree() <= minDegree {
			continue
		}
		removed[id] = true
		detach(l.Nodes[e.A], id)
		detach(l.Nodes[e.B], id)
	}
	if len(removed) == 0 {
		return 0
	}

	// Compact the edge slice and remap the IDs the nodes hold.
	remap := make(map[int]int, len(l.Edges))
	kept := l.Edges[:0]
	for _, e := range l.Edges {
		if removed[e.ID] {
			continue
		}
		remap[e.ID] = len(kept)
		e.ID = len(kept)
		kept = append(kept, e)
	}
	l.Edges = kept
	for _, n := range l.Nodes {
		for k, id := range n.Edges {
			n.Edges[k] = remap[id]
		}
	}
	return len(removed)
}

func detach(n *LatticeNode, edge int) {
	for k, id := range n.Edges {
		if id == edge {
			n.Edges = append(n.Edges[:k], n.Edges[k+1:]...)
			return
		}
	}
}

// perm returns the integers [0, n) in a random order, so thinning considers
// edges without bias toward one corner of the lattice.
func perm(rng *RNG, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// Other returns the node at the far end of edge from node.
func (l *Lattice) Other(edge *LatticeEdge, node int) int {
	if edge.A == node {
		return edge.B
	}
	return edge.A
}

func (l *Lattice) computeBounds() {
	l.Min, l.Max = l.Nodes[0].Pos, l.Nodes[0].Pos
	for _, n := range l.Nodes {
		l.Min.X, l.Min.Y = min(l.Min.X, n.Pos.X), min(l.Min.Y, n.Pos.Y)
		l.Max.X, l.Max.Y = max(l.Max.X, n.Pos.X), max(l.Max.Y, n.Pos.Y)
	}
}
