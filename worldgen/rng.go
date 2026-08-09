// Package worldgen generates the family's tile-grid dungeons: binary space
// partitioning carves rooms into leaves, L-shaped corridors join them into
// one connected map, short dead-end stubs add texture, and flood-fill
// utilities answer reachability and distance questions about the result.
//
// The tile grid itself belongs to each game; [Generate] digs through the
// [Carver] interface, tuned by a [Config] ([DefaultConfig] is the family's
// conventional constants), and returns the rooms as [geom.Rect] values.
// [FloodDist] produces a [Field] of step distances behind plain
// predicates, with [Reachable] and [StepsBetween] as shorthands and
// [Neighbors4] as the shared adjacency.
//
// Not every world is a dungeon. [NewLattice] generates the other shape the
// family builds on: a grid of nodes joined by axis-aligned edges with varied
// spacing, which a game reads as streets, districts or regions as it likes.
// [Lattice.Thin] then drops a share of those edges without stranding a node,
// which is what stops a generated grid looking like graph paper.
//
// Randomness flows from [RNG] (seeded by [NewRNG]), so a seed reproduces the
// same map everywhere.
package worldgen

import "math/rand/v2"

// RNG is the deterministic random source used across the family's
// generators, with the conventional integer-range and probability helpers.
type RNG struct {
	r *rand.Rand
}

// NewRNG returns a generator seeded the way the family's games seed their
// levels, so a seed reproduces the same map everywhere.
func NewRNG(seed int64) *RNG {
	u := uint64(seed)
	return &RNG{r: rand.New(rand.NewPCG(u, u^0x9e3779b97f4a7c15))}
}

// IntN returns a uniform int in [0, n).
func (g *RNG) IntN(n int) int {
	return g.r.IntN(n)
}

// Between returns a uniform int in [lo, hi]. When hi <= lo it returns lo.
func (g *RNG) Between(lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + g.r.IntN(hi-lo+1)
}

// BetweenF returns a uniform float64 in [lo, hi). When hi <= lo it returns
// lo.
func (g *RNG) BetweenF(lo, hi float64) float64 {
	if hi <= lo {
		return lo
	}
	return lo + g.r.Float64()*(hi-lo)
}

// Chance reports true with probability p.
func (g *RNG) Chance(p float64) bool {
	switch {
	case p <= 0:
		return false
	case p >= 1:
		return true
	default:
		return g.r.Float64() < p
	}
}
