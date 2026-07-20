package level

import (
	"errors"
	"fmt"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/worldgen"
)

// ErrUnreachable reports that no generation attempt produced a level the
// validator accepted.
var ErrUnreachable = errors.New("level: exhausted attempts producing a valid level")

// minDimension is the smallest usable grid edge.
const minDimension = 16

// GenerateConfig tunes [Generate]. The zero value (plus a size) selects
// the family's conventional constants.
type GenerateConfig struct {
	// Width and Height are the grid size in cells; both are raised to a
	// sane minimum.
	Width, Height int
	// Seed reproduces the level exactly.
	Seed int64
	// MaxAttempts caps how many layouts are tried before giving up.
	MaxAttempts int
	// Worldgen tunes the room-and-corridor digging.
	Worldgen worldgen.Config
}

func (c GenerateConfig) normalized() GenerateConfig {
	if c.Width < minDimension {
		c.Width = minDimension
	}
	if c.Height < minDimension {
		c.Height = minDimension
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 32
	}
	return c
}

// Pass furnishes a freshly dug level: heights, doors, vents, items —
// anything run between digging and validation. The engine's passes
// ([AssignHeights], [PlaceLowWalls], [PlaceLiftLedge], [PlaceDoors],
// [CarveVents], [AssignThemes], [AssignSky]) fit the signature via small
// closures, and games append their own.
type Pass func(l *Level, rng *worldgen.RNG, rooms []geom.Rect)

// Generate runs the family's level pipeline: dig rooms and corridors,
// place the spawn and the farthest exit, run the passes in order, and
// validate — retrying with a derived seed until the validator accepts or
// attempts run out. A nil validate accepts any level whose exit is
// reachable from its spawn.
//
// The same config always produces the same level.
func Generate(cfg GenerateConfig, passes []Pass, validate func(*Level) bool) (*Level, []geom.Rect, error) {
	cfg = cfg.normalized()
	if validate == nil {
		validate = func(l *Level) bool {
			return worldgen.Reachable(l.W, l.H, l.Spawn, l.Exit,
				func(c geom.Coord) bool { return l.Solid(c.X, c.Y) })
		}
	}
	for attempt := range cfg.MaxAttempts {
		// Derive a per-attempt seed deterministically from the base seed.
		sub := cfg.Seed + int64(attempt)*0x100000001b3
		l, rooms := generateOnce(cfg, sub, passes)
		l.Seed = cfg.Seed
		if validate(l) {
			return l, rooms, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: %dx%d seed=%d", ErrUnreachable, cfg.Width, cfg.Height, cfg.Seed)
}

func generateOnce(cfg GenerateConfig, seed int64, passes []Pass) (*Level, []geom.Rect) {
	l := New(cfg.Width, cfg.Height, seed)
	rng := worldgen.NewRNG(seed)
	rooms := worldgen.Generate(rng, l, cfg.Worldgen)
	PlaceSpawnExit(l, rooms)
	for _, pass := range passes {
		pass(l, rng, rooms)
	}
	return l, rooms
}

// PlaceSpawnExit puts the spawn at the first room's centre and the exit on
// the walkable cell farthest from it, so every run starts with the longest
// natural journey the layout offers.
func PlaceSpawnExit(l *Level, rooms []geom.Rect) {
	if len(rooms) == 0 {
		return
	}
	spawn := rooms[0].Center()
	field := worldgen.FloodDist(l.W, l.H, spawn,
		func(c geom.Coord) bool { return l.Solid(c.X, c.Y) }, nil)
	exit, best := spawn, 0
	for i, d := range field.D {
		if d > best {
			best = d
			exit = geom.Coord{X: i % l.W, Y: i / l.W}
		}
	}
	l.Spawn = spawn
	l.Set(spawn.X, spawn.Y, TileSpawn)
	if exit != spawn {
		l.Exit = exit
		l.Set(exit.X, exit.Y, TileExit)
	}
}
