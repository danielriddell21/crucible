package worldgen

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
)

// Axis names the direction a lattice edge runs in.
type Axis int

// The lattice axes.
const (
	// AxisX is an edge running along X, joining two nodes in the same row.
	AxisX Axis = iota
	// AxisY is an edge running along Y, joining two nodes in the same column.
	AxisY
)

// Plan is the infinite grid of lines a [Lattice] is cut from.
//
// Every line's position is a function of its index alone, so any part of the
// grid can be worked out without generating the rest of it. That is what lets
// a world grow: a patch generated an hour into a drive lines up exactly with
// one generated at the start, because neither depended on the other having
// happened. Indices are signed and unbounded in both directions.
//
// The alternative — walking outward adding a random gap each time — makes a
// line's position depend on every draw before it, so two patches generated
// independently disagree about where their shared edge is and the roads do not
// meet.
type Plan struct {
	seed uint64
	// nominal is the average gap between lines, jitter the most a line moves
	// from its nominal place. Two neighbouring lines can each move by jitter,
	// so a gap stays within [nominal-2*jitter, nominal+2*jitter].
	nominal, jitter float64
}

// NewPlan returns the grid of lines for a seed, with gaps between consecutive
// lines spread across [minSpan, maxSpan]. A range that is empty or inverted
// collapses to a uniform spacing.
func NewPlan(seed uint64, minSpan, maxSpan float64) Plan {
	minSpan = max(minSpan, 1)
	maxSpan = max(maxSpan, minSpan)
	return Plan{
		seed:    seed,
		nominal: (minSpan + maxSpan) / 2,
		jitter:  (maxSpan - minSpan) / 4,
	}
}

// X returns the position of vertical grid line i, for any signed i.
func (p Plan) X(i int) float64 { return p.line(0, i) }

// Y returns the position of horizontal grid line j, for any signed j.
func (p Plan) Y(j int) float64 { return p.line(1, j) }

func (p Plan) line(axis, i int) float64 {
	return float64(i)*p.nominal + p.jitter*signedUnit(p.seed, axis, i)
}

// Pos returns the position of the node at a cell.
func (p Plan) Pos(col, row int) geom.Vec2 {
	return geom.Vec2{X: p.X(col), Y: p.Y(row)}
}

// hash mixes a seed and two coordinates into a well-distributed integer. It is
// the whole basis of position-derived generation: every decision about a cell
// is taken from this rather than from a running stream, so it comes out the
// same whenever the cell is reached.
func hash(seed uint64, a, b int) uint64 {
	h := seed + 0x9e3779b97f4a7c15
	h ^= uint64(int64(a)) * 0xbf58476d1ce4e5b9
	h = (h ^ (h >> 30)) * 0xbf58476d1ce4e5b9
	h ^= uint64(int64(b)) * 0x94d049bb133111eb
	h = (h ^ (h >> 27)) * 0x94d049bb133111eb
	return h ^ (h >> 31)
}

// unit returns a value in [0, 1) derived from a seed and two coordinates.
func unit(seed uint64, a, b int) float64 {
	return float64(hash(seed, a, b)>>11) / float64(uint64(1)<<53)
}

// signedUnit returns a value in [-1, 1).
func signedUnit(seed uint64, a, b int) float64 { return unit(seed, a, b)*2 - 1 }

// LatticeNode is one junction of a [Lattice].
type LatticeNode struct {
	// ID indexes the node in [Lattice.Nodes]. IDs are never reused and never
	// renumbered, so anything holding one stays valid as the lattice grows.
	ID int
	// Pos is the node's position in world units.
	Pos geom.Vec2
	// Col and Row are the node's cell, which may be negative.
	Col, Row int
	// Edges holds the IDs of the edges meeting here.
	Edges []int
}

// Degree returns how many edges meet at the node.
func (n *LatticeNode) Degree() int { return len(n.Edges) }

// LatticeEdge joins two nodes of a [Lattice].
type LatticeEdge struct {
	// ID indexes the edge in [Lattice.Edges]. Like node IDs, stable for life.
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

// Lattice is a grid of nodes joined by axis-aligned edges, cut from a [Plan]
// and grown a region at a time.
//
// It is the shared skeleton under a street plan, a district map, an overworld
// of connected regions: the engine owns the topology and the geometry, and the
// game attaches its own meaning to it. Nothing here knows what an edge is for.
//
// Build one with [NewLattice] for a world of fixed size, or [NewGrowable]
// followed by [Lattice.Grow] for one that extends as far as anybody travels.
// Either way node and edge IDs are permanent: growth appends and never
// renumbers, so an agent holding an edge from an hour ago still holds it.
type Lattice struct {
	// Nodes and Edges are the graph. IDs index these slices.
	Nodes []*LatticeNode
	Edges []*LatticeEdge
	// Min and Max bound the node positions generated so far.
	Min, Max geom.Vec2

	plan  Plan
	thin  float64
	byRow map[[2]int]int   // cell to node ID
	built map[edgeKey]bool // edges already created
}

// LatticeConfig tunes lattice generation.
type LatticeConfig struct {
	// Cols and Rows are how many grid lines to lay out on each axis, for a
	// world of fixed size. [NewGrowable] ignores them.
	Cols, Rows int
	// MinSpan and MaxSpan bound the gap between consecutive grid lines, in
	// world units. Each gap varies independently, which is what stops the
	// result looking like graph paper.
	MinSpan, MaxSpan float64
	// Thin is the share of edges to drop, breaking up the regularity of the
	// grid. Zero keeps every edge.
	//
	// Which edges go is decided from their position rather than by shuffling,
	// and no node may lose more than one, so thinning never strands a junction
	// or leaves a dead end however the lattice was grown.
	Thin float64
}

// NewGrowable returns an empty lattice over a seed's plan. Nothing exists
// until [Lattice.Grow] is called.
func NewGrowable(seed uint64, cfg LatticeConfig) *Lattice {
	return &Lattice{
		plan:  NewPlan(seed, cfg.MinSpan, cfg.MaxSpan),
		thin:  min(max(cfg.Thin, 0), 1),
		byRow: map[[2]int]int{},
		built: map[edgeKey]bool{},
	}
}

// NewLattice builds a complete lattice of the configured size, centred on the
// origin, for a world that does not grow.
func NewLattice(seed uint64, cfg LatticeConfig) *Lattice {
	cols, rows := max(cfg.Cols, 2), max(cfg.Rows, 2)
	l := NewGrowable(seed, cfg)
	// Centred, so a camera starting at the origin starts amongst it.
	l.Grow(geom.Rect{X: -cols / 2, Y: -rows / 2, W: cols, H: rows})
	return l
}

// Grow generates every node and edge in a rectangle of cells that does not
// exist yet, and returns the IDs of the nodes it added. Cells already grown
// are left alone, so overlapping calls are cheap and repeatable.
//
// Edges are created between neighbouring cells only when both ends exist, and
// growing an adjoining region later fills in the edges that span the join.
func (l *Lattice) Grow(cells geom.Rect) []int {
	var added []int
	for col := cells.X; col < cells.X+cells.W; col++ {
		for row := cells.Y; row < cells.Y+cells.H; row++ {
			if id, ok := l.ensureNode(col, row); ok {
				added = append(added, id)
			}
		}
	}
	// Wire only after every node in the region exists, so an edge is never
	// missed because its far end had not been reached yet. The sweep starts
	// one cell short on each axis: those cells were grown earlier and left
	// their outward edges unmade because this region did not exist then, and
	// this is where the two get joined.
	for col := cells.X - 1; col < cells.X+cells.W; col++ {
		for row := cells.Y - 1; row < cells.Y+cells.H; row++ {
			l.wire(col, row)
		}
	}
	return added
}

// ensureNode creates the node for a cell if it is not already there, and
// reports whether it made one.
func (l *Lattice) ensureNode(col, row int) (int, bool) {
	key := [2]int{col, row}
	if id, ok := l.byRow[key]; ok {
		return id, false
	}
	n := &LatticeNode{ID: len(l.Nodes), Pos: l.plan.Pos(col, row), Col: col, Row: row}
	l.Nodes = append(l.Nodes, n)
	l.byRow[key] = n.ID
	l.note(n.Pos)
	return n.ID, true
}

// wire adds the edges leaving a cell toward increasing X and Y, where the far
// node exists and the edge is neither already built nor thinned away.
//
// Whether an edge exists is tracked per edge rather than per cell, because a
// cell can be half wired: grown while one neighbour existed and the other did
// not, then revisited later when the second arrives. Marking the cell done
// after the first visit would lose the second edge, and not marking it at all
// would build the first one twice.
func (l *Lattice) wire(col, row int) {
	from, ok := l.byRow[[2]int{col, row}]
	if !ok {
		return
	}
	for _, e := range [...]struct {
		dc, dr int
		axis   Axis
	}{{1, 0, AxisX}, {0, 1, AxisY}} {
		key := edgeKey{col, row, e.axis}
		if l.built[key] {
			continue
		}
		to, ok := l.byRow[[2]int{col + e.dc, row + e.dr}]
		if !ok {
			// The neighbour has not been reached yet; a later Grow that gets
			// there comes back and joins the two.
			continue
		}
		if l.removed(col, row, e.axis) {
			// Mark it anyway: the decision is positional and will not change,
			// so there is no point asking again.
			l.built[key] = true
			continue
		}
		l.addEdge(from, to, e.axis)
		l.built[key] = true
	}
}

// removed reports whether the edge leaving a cell along an axis was thinned
// away.
//
// The decision is local and positional: an edge is a candidate when its own
// hash falls under the share, and it is only actually removed when it is the
// strongest candidate among every edge touching either of its endpoints. That
// guarantees no node loses two edges — so nothing is ever stranded — without
// needing to see the whole lattice, which a growing world never can.
func (l *Lattice) removed(col, row int, axis Axis) bool {
	score, ok := l.candidate(col, row, axis)
	if !ok {
		return false
	}
	ends := [2][2]int{{col, row}, {col + 1, row}}
	if axis == AxisY {
		ends[1] = [2]int{col, row + 1}
	}
	for _, end := range ends {
		for _, other := range incident(end[0], end[1]) {
			if other == (edgeKey{col, row, axis}) {
				continue
			}
			if s, ok := l.candidate(other.col, other.row, other.axis); ok && s <= score {
				return false
			}
		}
	}
	return true
}

// edgeKey names an edge by the cell it leaves and the direction it runs.
type edgeKey struct {
	col, row int
	axis     Axis
}

// incident returns the four edges that could meet at a cell.
func incident(col, row int) [4]edgeKey {
	return [4]edgeKey{
		{col, row, AxisX},
		{col - 1, row, AxisX},
		{col, row, AxisY},
		{col, row - 1, AxisY},
	}
}

// candidate returns an edge's thinning score, and whether it is under the
// share at all. Scores are drawn from the edge's position, so two lattices
// grown in a different order agree about every one of them.
func (l *Lattice) candidate(col, row int, axis Axis) (float64, bool) {
	if l.thin <= 0 {
		return 0, false
	}
	// The axis is folded into the coordinate so the two edges leaving a cell
	// score differently.
	s := unit(l.plan.seed^0xa5a5a5a5, col*2+int(axis), row)
	return s, s < l.thin
}

func (l *Lattice) addEdge(a, b int, axis Axis) {
	na, nb := l.Nodes[a], l.Nodes[b]
	d := nb.Pos.Sub(na.Pos)
	e := &LatticeEdge{ID: len(l.Edges), A: a, B: b, Axis: axis, Dir: d.Normalize(), Length: d.Len()}
	l.Edges = append(l.Edges, e)
	na.Edges = append(na.Edges, e.ID)
	nb.Edges = append(nb.Edges, e.ID)
}

func (l *Lattice) note(p geom.Vec2) {
	if len(l.Nodes) == 1 {
		l.Min, l.Max = p, p
		return
	}
	l.Min.X, l.Min.Y = min(l.Min.X, p.X), min(l.Min.Y, p.Y)
	l.Max.X, l.Max.Y = max(l.Max.X, p.X), max(l.Max.Y, p.Y)
}

// Node returns the node at a cell, or nil where none has been grown.
func (l *Lattice) Node(col, row int) *LatticeNode {
	if id, ok := l.byRow[[2]int{col, row}]; ok {
		return l.Nodes[id]
	}
	return nil
}

// Other returns the node at the far end of edge from node.
func (l *Lattice) Other(edge *LatticeEdge, node int) int {
	if edge.A == node {
		return edge.B
	}
	return edge.A
}

// Cell returns the cell whose node lies nearest a world position. It is the
// inverse of [Plan.Pos], and what a caller uses to work out which part of the
// world to grow next.
func (l *Lattice) Cell(p geom.Vec2) (col, row int) {
	return l.plan.nearest(0, p.X), l.plan.nearest(1, p.Y)
}

// nearest inverts a line position. Jitter is bounded well below the nominal
// gap, so rounding the nominal index lands within one line of the answer and
// a short search settles it.
func (p Plan) nearest(axis int, v float64) int {
	guess := int(math.Round(v / p.nominal))
	best, bestD := guess, math.Abs(p.line(axis, guess)-v)
	for _, i := range [...]int{guess - 1, guess + 1} {
		if d := math.Abs(p.line(axis, i) - v); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// CellsAround returns the rectangle of cells covering a radius about a world
// position, for passing to [Lattice.Grow]. The rectangle always reaches past
// the radius rather than stopping inside it, so nothing within the radius is
// left ungrown.
func (l *Lattice) CellsAround(p geom.Vec2, radius float64) geom.Rect {
	radius = max(radius, 0)
	c0 := l.plan.before(0, p.X-radius)
	c1 := l.plan.after(0, p.X+radius)
	r0 := l.plan.before(1, p.Y-radius)
	r1 := l.plan.after(1, p.Y+radius)
	return geom.Rect{X: c0, Y: r0, W: c1 - c0 + 1, H: r1 - r0 + 1}
}

// before returns the largest line index at or below v, and after the smallest
// at or above it. Nearest can land the wrong side of the mark, which would
// leave a sliver of the requested area ungrown.
func (p Plan) before(axis int, v float64) int {
	i := p.nearest(axis, v)
	for p.line(axis, i) > v {
		i--
	}
	return i
}

func (p Plan) after(axis int, v float64) int {
	i := p.nearest(axis, v)
	for p.line(axis, i) < v {
		i++
	}
	return i
}
