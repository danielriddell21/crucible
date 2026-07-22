// Package ring is a fixed-capacity ring buffer for the rolling histories the
// family's sims keep: population counts, per-generation fitness, telemetry
// samples. Once full it overwrites its oldest entry, so it holds the most
// recent N values in constant memory.
//
// [Ring.Slice] returns the live contents oldest-first, ready to plot or log.
// The buffer is generic over the element type and imports nothing beyond the
// standard library, so it stays display-free and reusable across sims.
package ring

import "slices"

// Ring is a fixed-capacity buffer that retains the most recently pushed
// values, overwriting the oldest once it is full. The zero value is not
// usable; construct one with [New].
type Ring[T any] struct {
	buf  []T
	next int
	full bool
}

// New returns an empty ring that retains the last capacity values. A
// capacity below 1 is raised to 1.
func New[T any](capacity int) *Ring[T] {
	return &Ring[T]{buf: make([]T, max(capacity, 1))}
}

// Push appends a value, discarding the oldest once the ring is full.
func (r *Ring[T]) Push(v T) {
	r.buf[r.next] = v
	r.next = (r.next + 1) % len(r.buf)
	if r.next == 0 {
		r.full = true
	}
}

// Len reports how many values the ring currently holds, up to its capacity.
func (r *Ring[T]) Len() int {
	if r.full {
		return len(r.buf)
	}
	return r.next
}

// Cap reports the ring's fixed capacity.
func (r *Ring[T]) Cap() int { return len(r.buf) }

// Slice returns a copy of the retained values in insertion order, oldest
// first. The result is independent of the ring's internal storage.
func (r *Ring[T]) Slice() []T {
	if !r.full {
		return slices.Clone(r.buf[:r.next])
	}
	out := make([]T, 0, len(r.buf))
	out = append(out, r.buf[r.next:]...)
	out = append(out, r.buf[:r.next]...)
	return out
}
