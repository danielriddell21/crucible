package worldgen_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

// tiles is a minimal [worldgen.Carver]: a rectangular grid of open/solid
// cells that a game would back with its own tile type.
type tiles struct {
	w, h int
	open []bool
}

func (t *tiles) Width() int         { return t.w }
func (t *tiles) Height() int        { return t.h }
func (t *tiles) Open(x, y int) bool { return t.open[y*t.w+x] }
func (t *tiles) Carve(x, y int)     { t.open[y*t.w+x] = true }
func (t *tiles) solid(c geom.Coord) bool {
	if c.X < 0 || c.Y < 0 || c.X >= t.w || c.Y >= t.h {
		return true
	}
	return !t.open[c.Y*t.w+c.X]
}

// ExampleWeightedChoice picks a kind from a table of relative frequencies,
// the way a generator chooses which item or hazard to place.
func ExampleWeightedChoice() {
	rng := worldgen.NewRNG(1)
	weights := []float64{1, 3, 6} // common, uncommon, rare split 10/30/60

	var counts [3]int
	for range 1000 {
		counts[worldgen.WeightedChoice(rng, weights)]++
	}
	fmt.Println(counts[0] < counts[1] && counts[1] < counts[2])
	// Output: true
}

// ExampleGenerate carves a connected dungeon into a caller-owned grid and
// confirms the rooms reach one another. The same seed always produces the
// same map.
func ExampleGenerate() {
	grid := &tiles{w: 48, h: 40, open: make([]bool, 48*40)}
	rooms := worldgen.Generate(worldgen.NewRNG(1), grid, worldgen.Config{})

	first, last := rooms[0].Center(), rooms[len(rooms)-1].Center()
	fmt.Println("connected:", worldgen.Reachable(grid.w, grid.h, first, last, grid.solid))
	// Output: connected: true
}
