package level

import (
	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

// AssignThemes gives every room a random theme index in [0, numThemes),
// written to [Level.Theme] on each walkable cell, so renderers can vary
// wall texture per room. Corridor cells keep theme 0.
func AssignThemes(l *Level, rng *worldgen.RNG, rooms []geom.Rect, numThemes int) {
	if numThemes <= 0 {
		numThemes = 1
	}
	for _, r := range rooms {
		th := uint8(rng.IntN(numThemes))
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if l.At(x, y).Walkable() {
					l.Theme[l.Index(x, y)] = th
				}
			}
		}
	}
}

// SkyConfig tunes [AssignSky]. The zero value selects the family's
// conventional constants.
type SkyConfig struct {
	// Chance is the probability a room opens to the sky.
	Chance float64
	// Headroom is the floor-to-sky gap given to open-air cells.
	Headroom float64
}

// DefaultSkyConfig returns the constants the family's games open skies
// with.
func DefaultSkyConfig() SkyConfig {
	return SkyConfig{Chance: 0.3, Headroom: 3.0}
}

func (c SkyConfig) normalized() SkyConfig {
	d := DefaultSkyConfig()
	if c.Chance <= 0 {
		c.Chance = d.Chance
	}
	if c.Headroom <= 0 {
		c.Headroom = d.Headroom
	}
	return c
}

// AssignSky opens some rooms to the sky: their cells are marked in
// [Level.Sky], fully lit, and given tall headroom. Run it last — it is
// purely additive and leaves earlier passes untouched.
func AssignSky(l *Level, rng *worldgen.RNG, rooms []geom.Rect, cfg SkyConfig) {
	cfg = cfg.normalized()
	for _, r := range rooms {
		if !rng.Chance(cfg.Chance) {
			continue
		}
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if !l.At(x, y).Walkable() {
					continue
				}
				i := l.Index(x, y)
				l.Sky[i] = true
				l.Light[i] = 1
				l.SetCeil(x, y, l.Floor(x, y)+cfg.Headroom)
			}
		}
	}
}
