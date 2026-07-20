package level

import (
	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

// LowWallConfig tunes [PlaceLowWalls]. The zero value selects the family's
// conventional constants.
type LowWallConfig struct {
	// Max caps how many walls are lowered.
	Max int
	// Height is the half wall's top height in world units.
	Height float64
	// Chance is the probability an eligible divider is lowered.
	Chance float64
}

// DefaultLowWallConfig returns the constants the family's games place low
// walls with.
func DefaultLowWallConfig() LowWallConfig {
	return LowWallConfig{Max: 3, Height: 0.4, Chance: 0.4}
}

func (c LowWallConfig) normalized() LowWallConfig {
	d := DefaultLowWallConfig()
	if c.Max <= 0 {
		c.Max = d.Max
	}
	if c.Height <= 0 {
		c.Height = d.Height
	}
	if c.Chance <= 0 {
		c.Chance = d.Chance
	}
	return c
}

// PlaceLowWalls lowers a few clean wall dividers into see-over half walls
// by setting their wall-top height. A wall qualifies when it has open
// floor on exactly one opposing axis, so lowering it joins two rooms
// visually without exposing the surrounding solid mass.
func PlaceLowWalls(l *Level, rng *worldgen.RNG, cfg LowWallConfig) {
	cfg = cfg.normalized()
	placed := 0
	for y := 1; y < l.H-1 && placed < cfg.Max; y++ {
		for x := 1; x < l.W-1 && placed < cfg.Max; x++ {
			if l.At(x, y) != TileWall || l.WallTopH[l.Index(x, y)] != 0 {
				continue
			}
			if !dividesOpenSpace(l, geom.Coord{X: x, Y: y}) {
				continue
			}
			if !rng.Chance(cfg.Chance) {
				continue
			}
			l.WallTopH[l.Index(x, y)] = cfg.Height
			placed++
		}
	}
}

func dividesOpenSpace(l *Level, c geom.Coord) bool {
	openH := l.At(c.X-1, c.Y).Walkable() && l.At(c.X+1, c.Y).Walkable() &&
		l.At(c.X, c.Y-1) == TileWall && l.At(c.X, c.Y+1) == TileWall
	openV := l.At(c.X, c.Y-1).Walkable() && l.At(c.X, c.Y+1).Walkable() &&
		l.At(c.X-1, c.Y) == TileWall && l.At(c.X+1, c.Y) == TileWall
	return openH || openV
}

// DoorConfig tunes [PlaceDoors]. The zero value selects the family's
// conventional constants.
type DoorConfig struct {
	// Max caps how many doors are placed.
	Max int
	// Chance is the probability an eligible doorway takes a door.
	Chance float64
}

// DefaultDoorConfig returns the constants the family's games place doors
// with.
func DefaultDoorConfig() DoorConfig {
	return DoorConfig{Max: 8, Chance: 0.4}
}

func (c DoorConfig) normalized() DoorConfig {
	d := DefaultDoorConfig()
	if c.Max <= 0 {
		c.Max = d.Max
	}
	if c.Chance <= 0 {
		c.Chance = d.Chance
	}
	return c
}

// PlaceDoors turns a few clean doorways — floor cells walled on one axis
// and open on the other — into [TileDoor] cells, never adjacent to another
// door.
func PlaceDoors(l *Level, rng *worldgen.RNG, cfg DoorConfig) {
	cfg = cfg.normalized()
	placed := 0
	for y := 1; y < l.H-1 && placed < cfg.Max; y++ {
		for x := 1; x < l.W-1 && placed < cfg.Max; x++ {
			if l.At(x, y) != TileFloor || !doorway(l, x, y) {
				continue
			}
			if adjacentDoor(l, x, y) || !rng.Chance(cfg.Chance) {
				continue
			}
			l.Set(x, y, TileDoor)
			placed++
		}
	}
}

func doorway(l *Level, x, y int) bool {
	wallsX := l.At(x-1, y) == TileWall && l.At(x+1, y) == TileWall
	wallsY := l.At(x, y-1) == TileWall && l.At(x, y+1) == TileWall
	openX := l.At(x-1, y).Walkable() && l.At(x+1, y).Walkable()
	openY := l.At(x, y-1).Walkable() && l.At(x, y+1).Walkable()
	return (wallsX && openY) || (wallsY && openX)
}

func adjacentDoor(l *Level, x, y int) bool {
	for _, n := range worldgen.Neighbors4(geom.Coord{X: x, Y: y}) {
		if l.At(n.X, n.Y) == TileDoor {
			return true
		}
	}
	return false
}
