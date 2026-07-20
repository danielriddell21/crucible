package level

import (
	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

// Height-rule defaults shared by the family's movement and flood-fill
// checks.
const (
	// DefaultMaxStep is the largest floor rise an agent can climb in one
	// step.
	DefaultMaxStep = 0.3
	// DefaultMinHeadroom is the smallest floor-to-ceiling gap an agent
	// fits through.
	DefaultMinHeadroom = 0.8
)

// HeightsConfig tunes [AssignHeights]. The zero value selects the family's
// conventional constants.
type HeightsConfig struct {
	// Step is the world-unit rise of one height level.
	Step float64
	// MaxRoomRaise is the highest level a room may be raised to.
	MaxRoomRaise int
	// RaiseChance is the probability a room away from the spawn is
	// raised at all.
	RaiseChance float64
	// DaisRaise is how many levels the exit dais climbs above its room.
	DaisRaise int
	// SmoothingSweeps caps the relaxation passes that keep neighbouring
	// cells within one step of each other.
	SmoothingSweeps int
	// CorridorHeadroom is the floor-to-ceiling gap given to cells outside
	// any room.
	CorridorHeadroom float64
}

// DefaultHeightsConfig returns the constants the family's games generate
// with.
func DefaultHeightsConfig() HeightsConfig {
	return HeightsConfig{
		Step:             0.25,
		MaxRoomRaise:     3,
		RaiseChance:      0.35,
		DaisRaise:        2,
		SmoothingSweeps:  64,
		CorridorHeadroom: 0.9,
	}
}

func (c HeightsConfig) normalized() HeightsConfig {
	d := DefaultHeightsConfig()
	if c.Step <= 0 {
		c.Step = d.Step
	}
	if c.MaxRoomRaise <= 0 {
		c.MaxRoomRaise = d.MaxRoomRaise
	}
	if c.RaiseChance <= 0 {
		c.RaiseChance = d.RaiseChance
	}
	if c.DaisRaise <= 0 {
		c.DaisRaise = d.DaisRaise
	}
	if c.SmoothingSweeps <= 0 {
		c.SmoothingSweeps = d.SmoothingSweeps
	}
	if c.CorridorHeadroom <= 0 {
		c.CorridorHeadroom = d.CorridorHeadroom
	}
	return c
}

// AssignHeights raises some rooms onto higher levels, spreads those levels
// into the corridors, smooths every transition to single steps, lifts the
// exit onto a dais, and assigns ceilings with per-room headroom. Rooms are
// the rectangles [worldgen.Generate] returned; the spawn's room always
// stays at the base level.
func AssignHeights(l *Level, rng *worldgen.RNG, rooms []geom.Rect, cfg HeightsConfig) {
	cfg = cfg.normalized()
	lv := roomLevels(l, rng, rooms, cfg)
	spreadToCorridors(l, lv)
	smoothSteps(l, lv, cfg)
	raiseDais(l, lv, cfg)
	applyLevels(l, lv, cfg)
	assignCeilings(l, rng, rooms, cfg)
}

// StepOK reports whether an agent can move between two adjacent cells
// under the height rules: the rise must be climbable and the destination
// must leave headroom. Use it as the step gate of [worldgen.FloodDist];
// pass [DefaultMaxStep] and [DefaultMinHeadroom] for the conventional
// rules.
func (l *Level) StepOK(from, to geom.Coord, maxStep, minHeadroom float64) bool {
	const eps = 1e-9
	return l.Floor(to.X, to.Y)-l.Floor(from.X, from.Y) <= maxStep+eps &&
		l.Ceil(to.X, to.Y)-l.Floor(to.X, to.Y) >= minHeadroom-eps
}

func roomLevels(l *Level, rng *worldgen.RNG, rooms []geom.Rect, cfg HeightsConfig) []int {
	lv := make([]int, l.W*l.H)
	for i := range lv {
		lv[i] = -1 // unset
	}
	for _, r := range rooms {
		lvl := 0
		if !r.Contains(l.Spawn) && rng.Chance(cfg.RaiseChance) {
			lvl = rng.Between(1, cfg.MaxRoomRaise)
		}
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if l.At(x, y).Walkable() {
					lv[l.Index(x, y)] = lvl
				}
			}
		}
	}
	return lv
}

func spreadToCorridors(l *Level, lv []int) {
	var queue []geom.Coord
	for y := range l.H {
		for x := range l.W {
			if lv[l.Index(x, y)] >= 0 {
				queue = append(queue, geom.Coord{X: x, Y: y})
			}
		}
	}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, n := range worldgen.Neighbors4(c) {
			if !l.InBounds(n.X, n.Y) || !l.At(n.X, n.Y).Walkable() {
				continue
			}
			idx := l.Index(n.X, n.Y)
			if lv[idx] >= 0 {
				continue
			}
			lv[idx] = lv[l.Index(c.X, c.Y)]
			queue = append(queue, n)
		}
	}
	// Any walkable cell missed entirely sits at the base level.
	for i := range lv {
		if lv[i] < 0 {
			lv[i] = 0
		}
	}
}

func relaxLevels(l *Level, lv []int, sweeps int, adjust func(cur, neighbour int) (int, bool)) {
	for range sweeps {
		changed := false
		for y := range l.H {
			for x := range l.W {
				if relaxCell(l, lv, x, y, adjust) {
					changed = true
				}
			}
		}
		if !changed {
			return
		}
	}
}

func relaxCell(l *Level, lv []int, x, y int, adjust func(cur, neighbour int) (int, bool)) bool {
	if !l.At(x, y).Walkable() {
		return false
	}
	ci := l.Index(x, y)
	changed := false
	for _, n := range worldgen.Neighbors4(geom.Coord{X: x, Y: y}) {
		if !l.InBounds(n.X, n.Y) || !l.At(n.X, n.Y).Walkable() {
			continue
		}
		if nv, ok := adjust(lv[ci], lv[l.Index(n.X, n.Y)]); ok {
			lv[ci] = nv
			changed = true
		}
	}
	return changed
}

func smoothSteps(l *Level, lv []int, cfg HeightsConfig) {
	relaxLevels(l, lv, cfg.SmoothingSweeps, func(cur, neighbour int) (int, bool) {
		if cur > neighbour+1 {
			return neighbour + 1, true
		}
		return cur, false
	})
}

func raiseDais(l *Level, lv []int, cfg HeightsConfig) {
	e := l.Exit
	if !l.InBounds(e.X, e.Y) {
		return
	}
	lv[l.Index(e.X, e.Y)] += cfg.DaisRaise

	relaxLevels(l, lv, cfg.SmoothingSweeps, func(cur, neighbour int) (int, bool) {
		if cur < neighbour-1 {
			return neighbour - 1, true
		}
		return cur, false
	})
}

func applyLevels(l *Level, lv []int, cfg HeightsConfig) {
	for i, v := range lv {
		if l.Tiles[i].Walkable() {
			l.FloorH[i] = float64(v) * cfg.Step
		}
	}
}

func assignCeilings(l *Level, rng *worldgen.RNG, rooms []geom.Rect, cfg HeightsConfig) {
	inRoom := make([]bool, l.W*l.H)
	for _, r := range rooms {
		headroom := 1.0 + 0.2*float64(rng.IntN(4)) // 1.0 .. 1.6
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if !l.At(x, y).Walkable() {
					continue
				}
				inRoom[l.Index(x, y)] = true
				l.SetCeil(x, y, l.Floor(x, y)+headroom)
			}
		}
	}
	for y := range l.H {
		for x := range l.W {
			i := l.Index(x, y)
			if !l.Tiles[i].Walkable() || inRoom[i] {
				continue
			}
			l.SetCeil(x, y, l.Floor(x, y)+cfg.CorridorHeadroom)
		}
	}
}
