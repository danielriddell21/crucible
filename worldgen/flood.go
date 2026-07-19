package worldgen

import "github.com/danielriddell21/crucible/geom"

// Neighbors4 returns the four orthogonal neighbours of a cell.
func Neighbors4(c geom.Coord) [4]geom.Coord {
	return [4]geom.Coord{
		{X: c.X + 1, Y: c.Y},
		{X: c.X - 1, Y: c.Y},
		{X: c.X, Y: c.Y + 1},
		{X: c.X, Y: c.Y - 1},
	}
}

// Field is a per-cell distance field produced by FloodDist. Cells that were
// not reached hold -1.
type Field struct {
	W, H int
	D    []int
}

// At returns the distance to a cell, or -1 when it is out of bounds or was
// not reached.
func (f Field) At(c geom.Coord) int {
	if c.X < 0 || c.Y < 0 || c.X >= f.W || c.Y >= f.H {
		return -1
	}
	return f.D[c.Y*f.W+c.X]
}

// FloodDist breadth-first floods a w×h grid from src and returns the
// orthogonal step distance to every reachable cell. solid reports cells the
// flood may not enter. step, when non-nil, additionally gates each move
// from one cell to the next — the games use it for height rules such as
// "you can only climb so far in one step".
func FloodDist(w, h int, src geom.Coord, solid func(geom.Coord) bool, step func(from, to geom.Coord) bool) Field {
	f := Field{W: w, H: h, D: make([]int, w*h)}
	for i := range f.D {
		f.D[i] = -1
	}
	inBounds := func(c geom.Coord) bool { return c.X >= 0 && c.Y >= 0 && c.X < w && c.Y < h }
	if !inBounds(src) || solid(src) {
		return f
	}
	f.D[src.Y*w+src.X] = 0
	queue := []geom.Coord{src}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		base := f.D[c.Y*w+c.X]
		for _, n := range Neighbors4(c) {
			if !inBounds(n) || solid(n) {
				continue
			}
			if step != nil && !step(c, n) {
				continue
			}
			idx := n.Y*w + n.X
			if f.D[idx] != -1 {
				continue
			}
			f.D[idx] = base + 1
			queue = append(queue, n)
		}
	}
	return f
}

// Reachable reports whether dst can be reached from src on a w×h grid whose
// solid cells block movement.
func Reachable(w, h int, src, dst geom.Coord, solid func(geom.Coord) bool) bool {
	return FloodDist(w, h, src, solid, nil).At(dst) >= 0
}

// StepsBetween returns the orthogonal step distance from src to dst, or -1
// when dst is unreachable.
func StepsBetween(w, h int, src, dst geom.Coord, solid func(geom.Coord) bool) int {
	return FloodDist(w, h, src, solid, nil).At(dst)
}
