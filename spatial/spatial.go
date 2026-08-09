// Package spatial is the uniform grid the family's sims use to answer "what is
// near this point" without testing every candidate.
//
// A [Grid] buckets values by position into square cells. [Grid.Near] then
// visits only the cells a query circle touches, which turns an O(n) sweep over
// every agent, item or road into a walk over a handful of buckets. The grid is
// generic over what it holds and imports nothing beyond
// [github.com/danielriddell21/crucible/geom], so it stays display-free.
//
// Two shapes, because the family's worlds come in two:
//
//   - [New] builds an open-plane grid backed by a map, for a world with no
//     fixed extent. Cells materialise as they are used, so a sparse world
//     costs only what it occupies.
//   - [NewTorus] builds a fixed w×h grid backed by a slice, wrapping at every
//     edge. Reuse it across ticks with [Grid.Clear] and [Grid.Insert]: the
//     cells keep their capacity, so a rebuild every frame does not allocate.
//
// Choosing a cell size is the whole tuning story: roughly the radius of a
// typical query. Much smaller and a query walks many empty cells; much larger
// and each cell holds too many candidates to have narrowed anything.
package spatial

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
)

// Grid buckets values by position for radius queries. The zero value is not
// usable; build one with [New] or [NewTorus].
type Grid[T any] struct {
	cell float64

	// Exactly one of these is in use: cells for a torus, buckets for the open
	// plane. cols is zero for the open plane, which is what distinguishes them.
	cols, rows int
	cells      [][]T
	buckets    map[[2]int][]T

	n int
}

// New returns an open-plane grid with square cells of the given size, backed
// by a map. A cell size at or below zero is treated as one.
func New[T any](cell float64) *Grid[T] {
	return &Grid[T]{cell: positive(cell), buckets: map[[2]int][]T{}}
}

// NewTorus returns a grid covering [0, w) × [0, h) with square cells of the
// given size, wrapping at every edge: a query near one edge sees values near
// the opposite one, and a position outside the rectangle folds back into it.
//
// It is backed by a slice sized to the cell count, so it suits a world whose
// extent is fixed and whose contents are rebuilt every tick.
func NewTorus[T any](w, h, cell float64) *Grid[T] {
	cell = positive(cell)
	cols := max(1, int(math.Ceil(w/cell)))
	rows := max(1, int(math.Ceil(h/cell)))
	return &Grid[T]{cell: cell, cols: cols, rows: rows, cells: make([][]T, cols*rows)}
}

func positive(cell float64) float64 {
	if cell <= 0 {
		return 1
	}
	return cell
}

// Cell returns the grid's cell size.
func (g *Grid[T]) Cell() float64 { return g.cell }

// Len returns how many values the grid holds. A value inserted at several
// positions counts once per insertion.
func (g *Grid[T]) Len() int { return g.n }

// Insert files v under the cell containing p.
func (g *Grid[T]) Insert(p geom.Vec2, v T) {
	g.n++
	if g.cols == 0 {
		key := [2]int{floorDiv(p.X, g.cell), floorDiv(p.Y, g.cell)}
		g.buckets[key] = append(g.buckets[key], v)
		return
	}
	i := g.index(floorDiv(p.X, g.cell), floorDiv(p.Y, g.cell))
	g.cells[i] = append(g.cells[i], v)
}

// Clear empties the grid, keeping each cell's capacity so the next rebuild
// does not allocate.
func (g *Grid[T]) Clear() {
	g.n = 0
	if g.cols == 0 {
		for k, b := range g.buckets {
			g.buckets[k] = b[:0]
		}
		return
	}
	for i := range g.cells {
		g.cells[i] = g.cells[i][:0]
	}
}

// Near calls fn for every value in a cell the circle of the given radius
// around p touches. It is a broad phase, not an exact one: fn sees everything
// in those cells, including values beyond the radius, so callers still test
// the real distance. A value inserted into several cells is visited once per
// cell the query reaches.
func (g *Grid[T]) Near(p geom.Vec2, radius float64, fn func(T)) {
	radius = max(radius, 0)
	loX, hiX := floorDiv(p.X-radius, g.cell), floorDiv(p.X+radius, g.cell)
	loY, hiY := floorDiv(p.Y-radius, g.cell), floorDiv(p.Y+radius, g.cell)

	if g.cols == 0 {
		for cy := loY; cy <= hiY; cy++ {
			for cx := loX; cx <= hiX; cx++ {
				for _, v := range g.buckets[[2]int{cx, cy}] {
					fn(v)
				}
			}
		}
		return
	}

	// On a torus a query wider than the world would visit the same cell
	// repeatedly, so cap each span at one full lap.
	nx := min(hiX-loX+1, g.cols)
	ny := min(hiY-loY+1, g.rows)
	for dy := range ny {
		row := mod(loY+dy, g.rows)
		for dx := range nx {
			for _, v := range g.cells[row*g.cols+mod(loX+dx, g.cols)] {
				fn(v)
			}
		}
	}
}

func (g *Grid[T]) index(cx, cy int) int {
	return mod(cy, g.rows)*g.cols + mod(cx, g.cols)
}

// floorDiv returns the cell index for a coordinate, rounding toward negative
// infinity so cells stay uniform across the origin. A plain int conversion
// truncates toward zero, which would make the two cells either side of it
// share an index.
func floorDiv(v, cell float64) int { return int(math.Floor(v / cell)) }

func mod(x, n int) int {
	x %= n
	if x < 0 {
		x += n
	}
	return x
}

// InsertOnce files v under the cell containing p unless v is already the most
// recent value in that cell.
//
// It is for indexing a shape rather than a point: walking a road, a wall or a
// path and inserting at each step files it in every cell it crosses, and this
// skips the repeats where several steps land in the same one. It is a free
// function rather than a method because it needs a comparable element type,
// which a method cannot ask for on its own.
func InsertOnce[T comparable](g *Grid[T], p geom.Vec2, v T) {
	cx, cy := floorDiv(p.X, g.cell), floorDiv(p.Y, g.cell)
	var bucket []T
	if g.cols == 0 {
		bucket = g.buckets[[2]int{cx, cy}]
	} else {
		bucket = g.cells[g.index(cx, cy)]
	}
	if len(bucket) > 0 && bucket[len(bucket)-1] == v {
		return
	}
	g.Insert(p, v)
}
