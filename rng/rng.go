// Package rng seeds the family's deterministic random streams. A run is
// reproduced from a single integer seed, and unrelated systems draw from
// independent sub-streams so adding or reordering one never disturbs
// another's sequence.
//
// [Stream] is the shared primitive: a base seed plus a stream id selects an
// independent [math/rand/v2.Rand]. Generation-side code that wants the
// integer-range and probability helpers uses [worldgen.RNG] instead, which
// wraps the same PCG source.
package rng

import "math/rand/v2"

// Stream returns an independent deterministic RNG for the given stream of a
// seed. Different stream ids draw from non-overlapping sequences, so a
// game can give each subsystem — the level, the roster, generation number —
// its own reproducible stream from one seed.
func Stream(seed, stream uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, stream))
}
