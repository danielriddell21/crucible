package level_test

import (
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/level"
	"github.com/danielriddell21/crucible/worldgen"
)

func TestPlaceLowWalls(t *testing.T) {
	// Two rooms separated by a one-cell wall column: a clean divider.
	l := level.New(7, 5, 0)
	for y := 1; y <= 3; y++ {
		for x := 1; x <= 2; x++ {
			l.Carve(x, y)
		}
		for x := 4; x <= 5; x++ {
			l.Carve(x, y)
		}
	}

	level.PlaceLowWalls(l, worldgen.NewRNG(1), level.LowWallConfig{Chance: 1})
	cfg := level.DefaultLowWallConfig()
	lowered := 0
	for y := range l.H {
		for x := range l.W {
			if top := l.WallTop(x, y); top > 0 {
				lowered++
				if top != cfg.Height {
					t.Fatalf("wall top = %v", top)
				}
				if x != 3 {
					t.Fatalf("lowered wall at %d,%d is not on the divider", x, y)
				}
				if l.At(x, y) != level.TileWall {
					t.Fatal("lowered cell must stay a wall tile")
				}
			}
		}
	}
	if lowered == 0 || lowered > cfg.Max {
		t.Fatalf("lowered = %d", lowered)
	}
}

func TestPlaceLowWallsSkipsExposedPillar(t *testing.T) {
	// A free-standing pillar open on all four sides is not a clean divider
	// and must keep its full height.
	l := level.New(5, 5, 0)
	l.Carve(1, 2)
	l.Carve(3, 2)
	l.Carve(2, 1)
	l.Carve(2, 3)
	level.PlaceLowWalls(l, worldgen.NewRNG(1), level.LowWallConfig{Chance: 1})
	for i, top := range l.WallTopH {
		if top != 0 {
			t.Fatalf("wall %d lowered on an exposed pillar", i)
		}
	}
}

func TestPlaceDoors(t *testing.T) {
	// A corridor between two rooms: the corridor cells are doorways.
	l := level.New(9, 5, 0)
	for y := 1; y <= 3; y++ {
		for x := 1; x <= 2; x++ {
			l.Carve(x, y)
		}
		for x := 6; x <= 7; x++ {
			l.Carve(x, y)
		}
	}
	for x := 3; x <= 5; x++ {
		l.Carve(x, 2)
	}

	level.PlaceDoors(l, worldgen.NewRNG(1), level.DoorConfig{Chance: 1})
	doors := 0
	for y := range l.H {
		for x := range l.W {
			if l.At(x, y) != level.TileDoor {
				continue
			}
			doors++
			// Every door sits in a proper doorway: walls across one axis.
			wallsX := l.At(x-1, y) == level.TileWall && l.At(x+1, y) == level.TileWall
			wallsY := l.At(x, y-1) == level.TileWall && l.At(x, y+1) == level.TileWall
			if !wallsX && !wallsY {
				t.Fatalf("door at %d,%d not in a doorway", x, y)
			}
			for _, n := range worldgen.Neighbors4(geom.Coord{X: x, Y: y}) {
				if l.At(n.X, n.Y) == level.TileDoor {
					t.Fatalf("adjacent doors at %d,%d", x, y)
				}
			}
		}
	}
	if doors == 0 {
		t.Fatal("no doors placed in an obvious corridor")
	}
}

func TestPlaceDoorsRespectsCap(t *testing.T) {
	l, _ := generate(t, 5)
	level.PlaceDoors(l, worldgen.NewRNG(5), level.DoorConfig{Max: 2, Chance: 1})
	doors := 0
	for _, tile := range l.Tiles {
		if tile == level.TileDoor {
			doors++
		}
	}
	if doors > 2 {
		t.Fatalf("doors = %d, cap 2", doors)
	}
}
