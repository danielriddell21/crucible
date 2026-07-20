package level

import (
	"container/heap"
	"sort"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

// VentConfig tunes [CarveVents]. The zero value selects the family's
// conventional constants.
type VentConfig struct {
	// MaxNetworks caps how many separate vent networks are carved.
	MaxNetworks int
	// MaxMouths caps how many rooms one network opens into.
	MaxMouths int
	// DepthBias is the extra path cost per tunnel cell that touches
	// walkable ground, steering tunnels deep into the wall mass instead
	// of hugging room faces. Zero keeps the unbiased shortest paths.
	DepthBias int
}

// DefaultVentConfig returns the constants the family's games carve vents
// with.
func DefaultVentConfig() VentConfig {
	return VentConfig{MaxNetworks: 2, MaxMouths: 4, DepthBias: 2}
}

func (c VentConfig) normalized() VentConfig {
	d := DefaultVentConfig()
	if c.MaxNetworks <= 0 {
		c.MaxNetworks = d.MaxNetworks
	}
	if c.MaxMouths <= 0 {
		c.MaxMouths = d.MaxMouths
	}
	if c.DepthBias < 0 {
		c.DepthBias = 0
	}
	return c
}

type ventMouth struct {
	room int
	at   geom.Coord
}

// CarveVents digs crawlable [TileVent] networks through the level's wall
// masses, connecting several rooms per network, and refreshes
// [Level.VentMouths]. It is deterministic for a given level.
//
// Each mouth opens at the centre of the wall span its room shares with the
// network's wall mass, and mouths join the network by the cheapest path to
// the nearest already-carved tunnel, so networks branch like trees rather
// than snaking room to room. With a positive DepthBias the tunnels prefer
// to run deep inside the mass, staying hidden from the rooms they pass.
func CarveVents(l *Level, rooms []geom.Rect, cfg VentConfig) {
	cfg = cfg.normalized()
	comp := wallComponents(l)

	// For each component, one prospective mouth per room that borders it.
	mouths := map[int][]ventMouth{}
	for ri, r := range rooms {
		for id, cells := range perimeterTouches(l, comp, r) {
			mouths[id] = append(mouths[id], ventMouth{room: ri, at: cells[len(cells)/2]})
		}
	}

	// Rank components by how many rooms they touch, most first; ties break
	// on the smaller component id so the choice is deterministic.
	ids := make([]int, 0, len(mouths))
	for id, ms := range mouths {
		if len(ms) >= 2 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := ids[i], ids[j]
		if len(mouths[a]) != len(mouths[b]) {
			return len(mouths[a]) > len(mouths[b])
		}
		return a < b
	})

	networks := 0
	for _, id := range ids {
		if networks >= cfg.MaxNetworks {
			break
		}
		ms := mouths[id]
		sort.Slice(ms, func(i, j int) bool { return ms[i].room < ms[j].room })
		if len(ms) > cfg.MaxMouths {
			ms = spreadMouths(ms, cfg.MaxMouths)
		}
		if digNetwork(l, comp, id, ms, cfg.DepthBias) {
			networks++
		}
	}

	collectVentMouths(l)
}

// perimeterTouches groups the room's perimeter cells by the wall component
// they belong to, in perimeter order, so the middle cell of each group
// makes a centred mouth.
func perimeterTouches(l *Level, comp []int, r geom.Rect) map[int][]geom.Coord {
	touches := map[int][]geom.Coord{}
	for _, c := range roomPerimeter(r) {
		if !l.InBounds(c.X, c.Y) {
			continue
		}
		if id := comp[l.Index(c.X, c.Y)]; id >= 0 {
			touches[id] = append(touches[id], c)
		}
	}
	return touches
}

func spreadMouths(ms []ventMouth, n int) []ventMouth {
	out := make([]ventMouth, 0, n)
	for i := range n {
		out = append(out, ms[i*len(ms)/n])
	}
	return out
}

// digNetwork joins the mouths into one tunnel tree: each mouth digs the
// cheapest path to the nearest already-carved tunnel cell, so the network
// branches instead of snaking room to room.
func digNetwork(l *Level, comp []int, id int, ms []ventMouth, depthBias int) bool {
	inComp := func(c geom.Coord) bool {
		if !l.InBounds(c.X, c.Y) {
			return false
		}
		if comp[l.Index(c.X, c.Y)] == id && l.At(c.X, c.Y) == TileWall {
			return true
		}
		return l.At(c.X, c.Y) == TileVent
	}
	network := map[geom.Coord]bool{ms[0].at: true}
	joined := 1
	for _, m := range ms[1:] {
		path := cheapestPath(l, m.at, network, inComp, depthBias)
		if path == nil {
			continue
		}
		for _, c := range path {
			l.Set(c.X, c.Y, TileVent)
			network[c] = true
		}
		joined++
	}
	return joined >= 2
}

// cheapestPath runs Dijkstra from src across cells inComp allows until it
// reaches any cell of the network, preferring tunnels that keep away from
// walkable ground when depthBias is positive.
func cheapestPath(l *Level, src geom.Coord, network map[geom.Coord]bool, inComp func(geom.Coord) bool, depthBias int) []geom.Coord {
	if !inComp(src) {
		return nil
	}
	cellCost := func(c geom.Coord) int {
		cost := 1
		if depthBias > 0 && touchesGround(l, c) {
			cost += depthBias
		}
		return cost
	}

	dist := map[geom.Coord]int{src: cellCost(src)}
	prev := map[geom.Coord]geom.Coord{src: src}
	pq := &cellQueue{{at: src, cost: dist[src]}}
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(cellItem)
		if cur.cost > dist[cur.at] {
			continue // stale entry
		}
		if network[cur.at] {
			return walkBack(prev, cur.at)
		}
		for _, n := range worldgen.Neighbors4(cur.at) {
			if !inComp(n) {
				continue
			}
			nd := cur.cost + cellCost(n)
			if d, seen := dist[n]; seen && d <= nd {
				continue
			}
			dist[n] = nd
			prev[n] = cur.at
			heap.Push(pq, cellItem{at: n, cost: nd})
		}
	}
	return nil
}

// touchesGround reports whether a prospective tunnel cell is adjacent to
// walkable, non-vent ground — the cells a tunnel shows itself from.
func touchesGround(l *Level, c geom.Coord) bool {
	for _, n := range worldgen.Neighbors4(c) {
		if t := l.At(n.X, n.Y); t.Walkable() && t != TileVent {
			return true
		}
	}
	return false
}

func walkBack(prev map[geom.Coord]geom.Coord, end geom.Coord) []geom.Coord {
	var path []geom.Coord
	for c := end; ; c = prev[c] {
		path = append(path, c)
		if c == prev[c] {
			break
		}
	}
	return path
}

// wallComponents labels each interior wall cell with the id of the
// connected wall mass it belongs to; every other cell holds -1.
func wallComponents(l *Level) []int {
	comp := make([]int, l.W*l.H)
	for i := range comp {
		comp[i] = -1
	}
	next := 0
	for y := 1; y < l.H-1; y++ {
		for x := 1; x < l.W-1; x++ {
			if l.At(x, y) != TileWall || comp[l.Index(x, y)] >= 0 {
				continue
			}
			floodComponent(l, comp, geom.Coord{X: x, Y: y}, next)
			next++
		}
	}
	return comp
}

func floodComponent(l *Level, comp []int, start geom.Coord, id int) {
	comp[l.Index(start.X, start.Y)] = id
	queue := []geom.Coord{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range worldgen.Neighbors4(cur) {
			if n.X < 1 || n.Y < 1 || n.X >= l.W-1 || n.Y >= l.H-1 {
				continue
			}
			if l.At(n.X, n.Y) != TileWall || comp[l.Index(n.X, n.Y)] >= 0 {
				continue
			}
			comp[l.Index(n.X, n.Y)] = id
			queue = append(queue, n)
		}
	}
}

func roomPerimeter(r geom.Rect) []geom.Coord {
	out := make([]geom.Coord, 0, 2*r.W+2*r.H)
	for x := r.X; x < r.X+r.W; x++ {
		out = append(out, geom.Coord{X: x, Y: r.Y - 1})
	}
	for x := r.X; x < r.X+r.W; x++ {
		out = append(out, geom.Coord{X: x, Y: r.Y + r.H})
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		out = append(out, geom.Coord{X: r.X - 1, Y: y})
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		out = append(out, geom.Coord{X: r.X + r.W, Y: y})
	}
	return out
}

func collectVentMouths(l *Level) {
	l.VentMouths = l.VentMouths[:0]
	for y := range l.H {
		for x := range l.W {
			if l.At(x, y) != TileVent {
				continue
			}
			if touchesGround(l, geom.Coord{X: x, Y: y}) {
				l.VentMouths = append(l.VentMouths, geom.Coord{X: x, Y: y})
			}
		}
	}
}

type cellItem struct {
	at   geom.Coord
	cost int
}

// cellQueue is a deterministic min-heap: ties on cost break on Y then X so
// equal-cost paths always resolve the same way.
type cellQueue []cellItem

func (q cellQueue) Len() int { return len(q) }

func (q cellQueue) Less(i, j int) bool {
	if q[i].cost != q[j].cost {
		return q[i].cost < q[j].cost
	}
	if q[i].at.Y != q[j].at.Y {
		return q[i].at.Y < q[j].at.Y
	}
	return q[i].at.X < q[j].at.X
}

func (q cellQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *cellQueue) Push(x any) { *q = append(*q, x.(cellItem)) }

func (q *cellQueue) Pop() any {
	old := *q
	n := len(old)
	it := old[n-1]
	*q = old[:n-1]
	return it
}
