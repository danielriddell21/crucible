package worldgen

import "sort"

// WeightedChoice returns an index in [0, len(weights)) chosen with
// probability proportional to weights[i], drawing from rng. It returns -1
// when weights is empty or sums to zero. Negative weights are treated as
// zero, and zero-weight entries are never chosen.
//
// Games use it to pick a kind — an item, a hazard, a spawn — from a table
// of relative frequencies while a level is generated.
func WeightedChoice(rng *RNG, weights []float64) int {
	cum := make([]float64, len(weights))
	total := 0.0
	for i, w := range weights {
		if w > 0 {
			total += w
		}
		cum[i] = total
	}
	if total <= 0 {
		return -1
	}
	r := rng.BetweenF(0, total)
	// The smallest index whose cumulative weight exceeds r. Because r is
	// drawn from [0, total) and the final cumulative weight is total, this
	// is always a valid index, and it skips any zero-weight plateau.
	return sort.Search(len(cum), func(i int) bool { return cum[i] > r })
}
