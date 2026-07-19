package worldgen

import "github.com/danielriddell21/crucible/geom"

// Carver is the surface a generator digs a level into. Coordinates are
// always within [0, Width) × [0, Height). Open reports whether a cell has
// already been dug; Carve digs one.
type Carver interface {
	Width() int
	Height() int
	Open(x, y int) bool
	Carve(x, y int)
}

// Config tunes BSP generation. The zero value selects the family's
// conventional constants.
type Config struct {
	// MinLeaf is the smallest partition edge that may still be split.
	MinLeaf int
	// MaxLeaf is the edge length above which a partition must split.
	MaxLeaf int
	// MinRoom is the smallest room edge.
	MinRoom int
	// RoomPad keeps rooms this many cells inside their leaf.
	RoomPad int
	// MaxDepth caps the partition depth.
	MaxDepth int
	// StopChance is the probability that a small-enough partition stops
	// splitting early, for size variety.
	StopChance float64
	// StubAttempts is how many times to try carving a dead-end stub.
	StubAttempts int
	// MaxStubs caps the number of stubs carved.
	MaxStubs int
}

// DefaultConfig returns the constants the family's games generate with.
func DefaultConfig() Config {
	return Config{
		MinLeaf:      7,
		MaxLeaf:      16,
		MinRoom:      3,
		RoomPad:      1,
		MaxDepth:     7,
		StopChance:   0.3,
		StubAttempts: 24,
		MaxStubs:     4,
	}
}

func (c Config) normalized() Config {
	d := DefaultConfig()
	if c.MinLeaf <= 0 {
		c.MinLeaf = d.MinLeaf
	}
	if c.MaxLeaf <= 0 {
		c.MaxLeaf = d.MaxLeaf
	}
	if c.MinRoom <= 0 {
		c.MinRoom = d.MinRoom
	}
	if c.RoomPad <= 0 {
		c.RoomPad = d.RoomPad
	}
	if c.MaxDepth <= 0 {
		c.MaxDepth = d.MaxDepth
	}
	if c.StopChance <= 0 {
		c.StopChance = d.StopChance
	}
	if c.StubAttempts <= 0 {
		c.StubAttempts = d.StubAttempts
	}
	if c.MaxStubs <= 0 {
		c.MaxStubs = d.MaxStubs
	}
	return c
}

type bspNode struct {
	bounds      geom.Rect
	left, right *bspNode
	room        *geom.Rect
}

func (n *bspNode) leaf() bool {
	return n.left == nil && n.right == nil
}

type generator struct {
	rng *RNG
	cfg Config
	c   Carver
}

// Generate partitions the carver's area, digs one room per leaf, joins
// sibling partitions with L-shaped corridors, adds dead-end stubs, and
// returns the rooms. The same seed always produces the same map.
func Generate(rng *RNG, c Carver, cfg Config) []geom.Rect {
	g := &generator{rng: rng, cfg: cfg.normalized(), c: c}
	root := &bspNode{bounds: geom.Rect{W: c.Width(), H: c.Height()}}
	g.split(root, 0)
	g.carveRooms(root)
	g.connect(root)
	g.carveStubs()
	return collectRooms(root)
}

func (g *generator) split(n *bspNode, depth int) {
	if depth >= g.cfg.MaxDepth {
		return
	}
	w, h := n.bounds.W, n.bounds.H
	canV := w >= 2*g.cfg.MinLeaf // vertical cut -> left/right
	canH := h >= 2*g.cfg.MinLeaf // horizontal cut -> top/bottom
	if !canV && !canH {
		return
	}
	// Small enough regions stop splitting most of the time, for size variety.
	// The root always splits so even small maps contain more than one room.
	if depth > 0 && w <= g.cfg.MaxLeaf && h <= g.cfg.MaxLeaf && g.rng.Chance(g.cfg.StopChance) {
		return
	}

	vertical := g.chooseSplitAxis(w, h, canV, canH)
	if vertical {
		at := g.rng.Between(g.cfg.MinLeaf, w-g.cfg.MinLeaf)
		n.left = &bspNode{bounds: geom.Rect{X: n.bounds.X, Y: n.bounds.Y, W: at, H: h}}
		n.right = &bspNode{bounds: geom.Rect{X: n.bounds.X + at, Y: n.bounds.Y, W: w - at, H: h}}
	} else {
		at := g.rng.Between(g.cfg.MinLeaf, h-g.cfg.MinLeaf)
		n.left = &bspNode{bounds: geom.Rect{X: n.bounds.X, Y: n.bounds.Y, W: w, H: at}}
		n.right = &bspNode{bounds: geom.Rect{X: n.bounds.X, Y: n.bounds.Y + at, W: w, H: h - at}}
	}
	g.split(n.left, depth+1)
	g.split(n.right, depth+1)
}

func (g *generator) chooseSplitAxis(w, h int, canV, canH bool) bool {
	switch {
	case canV && !canH:
		return true
	case canH && !canV:
		return false
	case w > h:
		return true
	case h > w:
		return false
	default:
		return g.rng.Chance(0.5)
	}
}

func (g *generator) carveRooms(n *bspNode) {
	if !n.leaf() {
		if n.left != nil {
			g.carveRooms(n.left)
		}
		if n.right != nil {
			g.carveRooms(n.right)
		}
		return
	}
	b := n.bounds
	maxW, maxH := b.W-2*g.cfg.RoomPad, b.H-2*g.cfg.RoomPad
	if maxW < g.cfg.MinRoom || maxH < g.cfg.MinRoom {
		return
	}
	rw := g.rng.Between(g.cfg.MinRoom, maxW)
	rh := g.rng.Between(g.cfg.MinRoom, maxH)
	rx := b.X + g.cfg.RoomPad + g.rng.IntN(maxW-rw+1)
	ry := b.Y + g.cfg.RoomPad + g.rng.IntN(maxH-rh+1)
	room := geom.Rect{X: rx, Y: ry, W: rw, H: rh}
	n.room = &room
	for y := ry; y < ry+rh; y++ {
		for x := rx; x < rx+rw; x++ {
			g.c.Carve(x, y)
		}
	}
}

func (g *generator) connect(n *bspNode) *geom.Rect {
	if n.room != nil {
		return n.room
	}
	var lr, rr *geom.Rect
	if n.left != nil {
		lr = g.connect(n.left)
	}
	if n.right != nil {
		rr = g.connect(n.right)
	}
	if lr != nil && rr != nil {
		g.carveCorridor(lr.Center(), rr.Center())
	}
	if lr != nil {
		return lr
	}
	return rr
}

func (g *generator) carveCorridor(a, b geom.Coord) {
	if g.rng.Chance(0.5) {
		g.carveH(a.X, b.X, a.Y)
		g.carveV(a.Y, b.Y, b.X)
	} else {
		g.carveV(a.Y, b.Y, a.X)
		g.carveH(a.X, b.X, b.Y)
	}
}

func (g *generator) carveH(x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		if !g.c.Open(x, y) {
			g.c.Carve(x, y)
		}
	}
}

func (g *generator) carveV(y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		if !g.c.Open(x, y) {
			g.c.Carve(x, y)
		}
	}
}

func (g *generator) carveStubs() {
	w, h := g.c.Width(), g.c.Height()
	carved := 0
	for range g.cfg.StubAttempts {
		if carved >= g.cfg.MaxStubs {
			return
		}
		x := g.rng.Between(1, w-2)
		y := g.rng.Between(1, h-2)
		if !g.c.Open(x, y) {
			continue
		}
		dirs := [4]geom.Coord{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
		d := dirs[g.rng.IntN(len(dirs))]
		length := g.rng.Between(1, 3)
		if g.digStub(geom.Coord{X: x, Y: y}, d, length) {
			carved++
		}
	}
}

func (g *generator) digStub(f, d geom.Coord, length int) bool {
	w, h := g.c.Width(), g.c.Height()
	dug := 0
	cell := f
	for range length {
		cell = cell.Add(d)
		if cell.X < 1 || cell.Y < 1 || cell.X >= w-1 || cell.Y >= h-1 {
			break
		}
		if g.c.Open(cell.X, cell.Y) {
			break
		}
		open := 0
		for _, n := range Neighbors4(cell) {
			if g.c.Open(n.X, n.Y) {
				open++
			}
		}
		if open != 1 {
			break
		}
		g.c.Carve(cell.X, cell.Y)
		dug++
	}
	return dug > 0
}

func collectRooms(n *bspNode) []geom.Rect {
	if n == nil {
		return nil
	}
	if n.room != nil {
		return []geom.Rect{*n.room}
	}
	return append(collectRooms(n.left), collectRooms(n.right)...)
}
