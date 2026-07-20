package level_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/level"
	"github.com/danielriddell21/crucible/worldgen"
)

// ExampleGenerate runs the family's full level pipeline: dig rooms and
// corridors, place the spawn and the farthest exit, then furnish the
// layout with heights, half walls, doors, vents, themes, and sky. A game
// appends its own passes for items and markers, and its own validator.
func ExampleGenerate() {
	passes := []level.Pass{
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.AssignHeights(l, rng, rooms, level.HeightsConfig{})
			level.PlaceLowWalls(l, rng, level.LowWallConfig{})
			level.PlaceDoors(l, rng, level.DoorConfig{})
			level.PlaceLiftLedge(l, rng, level.LiftConfig{}, nil)
			level.CarveVents(l, rooms, level.VentConfig{})
			level.AssignThemes(l, rng, rooms, 3)
			level.AssignSky(l, rng, rooms, level.SkyConfig{})
		},
	}

	l, rooms, err := level.Generate(level.GenerateConfig{Width: 48, Height: 40, Seed: 7}, passes, nil)
	if err != nil {
		fmt.Println("generate:", err)
		return
	}

	fmt.Println("rooms:", len(rooms) > 1)
	fmt.Println("spawn floor:", l.Floor(l.Spawn.X, l.Spawn.Y))
	fmt.Println("exit raised:", l.Floor(l.Exit.X, l.Exit.Y) > 0)
	// Output:
	// rooms: true
	// spawn floor: 0
	// exit raised: true
}

// ExampleLevel_StepOK gates a flood fill with the height rules, so
// reachability respects what an agent can actually climb.
func ExampleLevel_StepOK() {
	l := level.New(3, 1, 0)
	for x := range 3 {
		l.Carve(x, 0)
	}
	l.SetFloor(2, 0, 1.0) // a full-unit rise: too tall to climb

	field := worldgen.FloodDist(l.W, l.H, l.Spawn,
		func(c geom.Coord) bool { return l.Solid(c.X, c.Y) },
		func(from, to geom.Coord) bool {
			return l.StepOK(from, to, level.DefaultMaxStep, level.DefaultMinHeadroom)
		})
	fmt.Println(field.At(geom.Coord{X: 1, Y: 0}), field.At(geom.Coord{X: 2, Y: 0}))
	// Output: 1 -1
}
