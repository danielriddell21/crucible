package level_test

import (
	"strings"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/level"
	"github.com/danielriddell21/crucible/raycast"
	"github.com/danielriddell21/crucible/worldgen"
)

func TestTileProperties(t *testing.T) {
	walkable := []level.Tile{
		level.TileFloor, level.TileDoor, level.TileVent,
		level.TileCover, level.TileSpawn, level.TileExit,
	}
	for _, tile := range walkable {
		if !tile.Walkable() {
			t.Errorf("%c must be walkable", tile.Rune())
		}
	}
	for _, tile := range []level.Tile{level.TileWall, level.TileSwitch} {
		if tile.Walkable() {
			t.Errorf("%c must not be walkable", tile.Rune())
		}
	}
	seen := map[rune]bool{}
	all := []level.Tile{
		level.TileFloor, level.TileWall, level.TileDoor, level.TileVent,
		level.TileSwitch, level.TileCover, level.TileSpawn, level.TileExit,
	}
	for _, tile := range all {
		r := tile.Rune()
		if seen[r] {
			t.Errorf("rune %c reused", r)
		}
		seen[r] = true
	}
}

func TestNewDefaults(t *testing.T) {
	l := level.New(8, 6, 42)
	if l.W != 8 || l.H != 6 || l.Seed != 42 {
		t.Fatalf("level = %dx%d seed %d", l.W, l.H, l.Seed)
	}
	if l.At(3, 3) != level.TileWall {
		t.Fatal("new level must be solid wall")
	}
	if l.Floor(3, 3) != 0 || l.Ceil(3, 3) != 1 || l.LightAt(3, 3) != 1 {
		t.Fatal("default heights/light wrong")
	}
	// Out of bounds reads are safe and solid.
	if l.At(-1, 0) != level.TileWall || !l.Solid(99, 99) {
		t.Fatal("out of bounds must read as wall")
	}
	if l.Floor(-1, 0) != 0 || l.Ceil(-1, 0) != 1 || l.LightAt(-1, 0) != 1 || l.SkyAt(-1, 0) {
		t.Fatal("out of bounds height/light defaults wrong")
	}
	l.Set(-1, 0, level.TileFloor) // must not panic
	l.SetFloor(-1, 0, 5)          // must not panic
	l.SetCeil(99, 99, 5)          // must not panic
	if l.WallTop(-1, 0) != 0 {
		t.Fatal("out of bounds wall top must be 0")
	}
}

// generate digs a standard dungeon into a fresh level.
func generate(t *testing.T, seed int64) (*level.Level, []geom.Rect) {
	t.Helper()
	l := level.New(48, 40, seed)
	rooms := worldgen.Generate(worldgen.NewRNG(seed), l, worldgen.Config{})
	if len(rooms) < 3 {
		t.Fatalf("only %d rooms", len(rooms))
	}
	l.Spawn = rooms[0].Center()
	l.Exit = rooms[len(rooms)-1].Center()
	l.Set(l.Spawn.X, l.Spawn.Y, level.TileSpawn)
	l.Set(l.Exit.X, l.Exit.Y, level.TileExit)
	return l, rooms
}

func TestLevelIsACarver(t *testing.T) {
	l, _ := generate(t, 1)
	open := 0
	for y := range l.H {
		for x := range l.W {
			if l.Open(x, y) {
				open++
			}
		}
	}
	if open == 0 {
		t.Fatal("Generate carved nothing into the level")
	}
	dump := l.String()
	if !strings.Contains(dump, "#") || !strings.Contains(dump, ".") {
		t.Fatal("String dump missing walls or floor")
	}
	if len(strings.Split(dump, "\n")) != l.H {
		t.Fatal("String dump has wrong row count")
	}
}

func TestStepOK(t *testing.T) {
	l := level.New(4, 1, 0)
	for x := range 4 {
		l.Carve(x, 0)
	}
	l.SetFloor(1, 0, 0.25)
	l.SetCeil(1, 0, 1.25) // ceilings rise with their floors
	l.SetFloor(2, 0, 1.0)
	l.SetCeil(2, 0, 2.0)
	l.SetCeil(3, 0, 0.5) // crawl space too low

	from := geom.Coord{X: 0, Y: 0}
	if !l.StepOK(from, geom.Coord{X: 1, Y: 0}, level.DefaultMaxStep, level.DefaultMinHeadroom) {
		t.Error("single step must be climbable")
	}
	if l.StepOK(from, geom.Coord{X: 2, Y: 0}, level.DefaultMaxStep, level.DefaultMinHeadroom) {
		t.Error("a full-unit rise must not be climbable")
	}
	if l.StepOK(geom.Coord{X: 2, Y: 0}, geom.Coord{X: 3, Y: 0}, level.DefaultMaxStep, level.DefaultMinHeadroom) {
		t.Error("no headroom must block the step")
	}
	// Dropping down is always allowed.
	if !l.StepOK(geom.Coord{X: 2, Y: 0}, geom.Coord{X: 1, Y: 0}, level.DefaultMaxStep, level.DefaultMinHeadroom) {
		t.Error("dropping down must be allowed")
	}
}

// A Level must satisfy the raycast height interface so it plugs straight
// into the column walker.
var _ raycast.Heights = (*level.Level)(nil)
