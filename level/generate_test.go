package level_test

import (
	"errors"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/level"
	"github.com/danielriddell21/crucible/worldgen"
)

// standardPasses is the full furnishing pipeline the games run.
func standardPasses() []level.Pass {
	return []level.Pass{
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.AssignHeights(l, rng, rooms, level.HeightsConfig{})
		},
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.PlaceLowWalls(l, rng, level.LowWallConfig{})
		},
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.PlaceDoors(l, rng, level.DoorConfig{})
		},
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.CarveVents(l, rooms, level.VentConfig{})
		},
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.AssignThemes(l, rng, rooms, 3)
		},
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.AssignSky(l, rng, rooms, level.SkyConfig{})
		},
	}
}

func TestGeneratePipeline(t *testing.T) {
	cfg := level.GenerateConfig{Width: 48, Height: 40, Seed: 11}
	l, rooms, err := level.Generate(cfg, standardPasses(), nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(rooms) < 2 {
		t.Fatalf("rooms = %d", len(rooms))
	}
	if l.At(l.Spawn.X, l.Spawn.Y) != level.TileSpawn || l.At(l.Exit.X, l.Exit.Y) != level.TileExit {
		t.Fatal("spawn/exit tiles not placed")
	}
	solid := func(c geom.Coord) bool { return l.Solid(c.X, c.Y) }
	if !worldgen.Reachable(l.W, l.H, l.Spawn, l.Exit, solid) {
		t.Fatal("exit unreachable")
	}
}

func TestGenerateDeterministic(t *testing.T) {
	cfg := level.GenerateConfig{Width: 48, Height: 40, Seed: 12}
	a, _, err := level.Generate(cfg, standardPasses(), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := level.Generate(cfg, standardPasses(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatal("same config produced different levels")
	}
	for i := range a.FloorH {
		if a.FloorH[i] != b.FloorH[i] || a.Theme[i] != b.Theme[i] || a.Sky[i] != b.Sky[i] {
			t.Fatal("same config produced different furnishing")
		}
	}
}

func TestGenerateRetriesUntilValid(t *testing.T) {
	attempts := 0
	validate := func(*level.Level) bool {
		attempts++
		return attempts >= 3
	}
	_, _, err := level.Generate(level.GenerateConfig{Width: 20, Height: 20, Seed: 1}, nil, validate)
	if err != nil || attempts != 3 {
		t.Fatalf("err = %v, attempts = %d", err, attempts)
	}
}

func TestGenerateExhaustsAttempts(t *testing.T) {
	never := func(*level.Level) bool { return false }
	_, _, err := level.Generate(level.GenerateConfig{Width: 20, Height: 20, Seed: 1, MaxAttempts: 4}, nil, never)
	if !errors.Is(err, level.ErrUnreachable) {
		t.Fatalf("err = %v", err)
	}
}

func TestGenerateExitIsFar(t *testing.T) {
	l, _, err := level.Generate(level.GenerateConfig{Width: 48, Height: 40, Seed: 13}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	solid := func(c geom.Coord) bool { return l.Solid(c.X, c.Y) }
	field := worldgen.FloodDist(l.W, l.H, l.Spawn, solid, nil)
	exitDist := field.At(l.Exit)
	for _, d := range field.D {
		if d > exitDist {
			t.Fatalf("a cell lies farther (%d) than the exit (%d)", d, exitDist)
		}
	}
}

func TestAssignThemesPerRoom(t *testing.T) {
	l, rooms := generate(t, 21)
	level.AssignThemes(l, worldgen.NewRNG(21), rooms, 3)
	for _, r := range rooms {
		c := r.Center()
		th := l.ThemeAt(c.X, c.Y)
		// A room is uniformly themed.
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if l.At(x, y).Walkable() && l.ThemeAt(x, y) != th {
					t.Fatalf("room at %v mixes themes", c)
				}
			}
		}
	}
	if l.ThemeAt(-1, 0) != 0 {
		t.Fatal("out of bounds theme must be 0")
	}
}

func TestAssignSky(t *testing.T) {
	opened := false
	for seed := int64(1); seed <= 5 && !opened; seed++ {
		l, rooms := generate(t, seed)
		level.AssignHeights(l, worldgen.NewRNG(seed), rooms, level.HeightsConfig{})
		level.AssignSky(l, worldgen.NewRNG(seed), rooms, level.SkyConfig{})
		cfg := level.DefaultSkyConfig()
		for y := range l.H {
			for x := range l.W {
				if !l.SkyAt(x, y) {
					continue
				}
				opened = true
				if l.LightAt(x, y) != 1 {
					t.Fatal("sky cell must be fully lit")
				}
				if gap := l.Ceil(x, y) - l.Floor(x, y); gap != cfg.Headroom {
					t.Fatalf("sky headroom = %v", gap)
				}
				if !l.At(x, y).Walkable() {
					t.Fatal("sky opened over a wall")
				}
			}
		}
	}
	if !opened {
		t.Fatal("no room opened to the sky across five seeds")
	}
}
