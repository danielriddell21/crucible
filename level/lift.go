package level

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

// Lift is a platform that cycles between two floor heights.
type Lift struct {
	Low, High float64
}

// LiftHeight returns the platform height of lf at time t seconds, cycling
// dwell seconds of rest at each end with travel seconds of movement
// between them.
func LiftHeight(lf Lift, t, dwell, travel float64) float64 {
	period := 2 * (dwell + travel)
	p := math.Mod(t, period)
	switch {
	case p < dwell:
		return lf.Low
	case p < dwell+travel:
		return lf.Low + (lf.High-lf.Low)*(p-dwell)/travel
	case p < 2*dwell+travel:
		return lf.High
	default:
		return lf.High - (lf.High-lf.Low)*(p-2*dwell-travel)/travel
	}
}

// LiftConfig tunes [PlaceLiftLedge]. The zero value selects the family's
// conventional constants.
type LiftConfig struct {
	// Raise is how many height steps the ledge sits above its neck.
	Raise int
	// Step is the world-unit rise of one height step; match the value
	// given to [AssignHeights].
	Step float64
	// Chance is the probability an eligible dead end takes the ledge.
	Chance float64
}

// DefaultLiftConfig returns the constants the family's games place lifts
// with.
func DefaultLiftConfig() LiftConfig {
	return LiftConfig{Raise: 3, Step: 0.25, Chance: 0.5}
}

func (c LiftConfig) normalized() LiftConfig {
	d := DefaultLiftConfig()
	if c.Raise <= 0 {
		c.Raise = d.Raise
	}
	if c.Step <= 0 {
		c.Step = d.Step
	}
	if c.Chance <= 0 {
		c.Chance = d.Chance
	}
	return c
}

// PlaceLiftLedge turns at most one dead-end floor cell into a raised
// reward ledge served by a lift on the cell before it, and returns the
// ledge cell so the game can place its reward there. avoid, when non-nil,
// vetoes cells the game has already claimed (items, secrets). It reports
// false when no eligible dead end took a ledge.
func PlaceLiftLedge(l *Level, rng *worldgen.RNG, cfg LiftConfig, avoid func(geom.Coord) bool) (geom.Coord, bool) {
	cfg = cfg.normalized()
	if avoid == nil {
		avoid = func(geom.Coord) bool { return false }
	}
	for y := range l.H {
		for x := range l.W {
			if c := (geom.Coord{X: x, Y: y}); tryLiftLedge(l, rng, cfg, avoid, c) {
				return c, true
			}
		}
	}
	return geom.Coord{}, false
}

func tryLiftLedge(l *Level, rng *worldgen.RNG, cfg LiftConfig, avoid func(geom.Coord) bool, d geom.Coord) bool {
	if l.At(d.X, d.Y) != TileFloor || avoid(d) {
		return false
	}
	nb := walkableNeighbors(l, d)
	if len(nb) != 1 { // ledges grow only from dead ends
		return false
	}
	neck := nb[0]
	if l.At(neck.X, neck.Y) != TileFloor || avoid(neck) || neck == l.Spawn || neck == l.Exit {
		return false
	}
	if !rng.Chance(cfg.Chance) {
		return false
	}
	low := l.Floor(neck.X, neck.Y)
	high := low + float64(cfg.Raise)*cfg.Step
	l.SetFloor(d.X, d.Y, high)
	// Both the ledge and the lift shaft need headroom above the raised
	// platform, not just above their static floors.
	l.SetCeil(d.X, d.Y, high+1)
	l.SetCeil(neck.X, neck.Y, high+1)
	if l.Lifts == nil {
		l.Lifts = make(map[geom.Coord]Lift)
	}
	l.Lifts[neck] = Lift{Low: low, High: high}
	return true
}

func walkableNeighbors(l *Level, c geom.Coord) []geom.Coord {
	var out []geom.Coord
	for _, n := range worldgen.Neighbors4(c) {
		if l.InBounds(n.X, n.Y) && l.At(n.X, n.Y).Walkable() {
			out = append(out, n)
		}
	}
	return out
}
