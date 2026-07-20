package level_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/level"
	"github.com/danielriddell21/crucible/worldgen"
)

func heightsLevel(t *testing.T, seed int64) (*level.Level, []geom.Rect) {
	t.Helper()
	l, rooms := generate(t, seed)
	level.AssignHeights(l, worldgen.NewRNG(seed), rooms, level.HeightsConfig{})
	return l, rooms
}

func TestAssignHeightsSmoothSteps(t *testing.T) {
	l, _ := heightsLevel(t, 3)
	cfg := level.DefaultHeightsConfig()
	const eps = 1e-9
	for y := range l.H {
		for x := range l.W {
			if !l.At(x, y).Walkable() {
				continue
			}
			for _, n := range worldgen.Neighbors4(geom.Coord{X: x, Y: y}) {
				if !l.InBounds(n.X, n.Y) || !l.At(n.X, n.Y).Walkable() {
					continue
				}
				rise := l.Floor(n.X, n.Y) - l.Floor(x, y)
				if rise > cfg.Step+eps {
					t.Fatalf("unclimbable rise %v at %d,%d -> %v", rise, x, y, n)
				}
			}
		}
	}
}

func TestAssignHeightsSpawnAtBaseExitRaised(t *testing.T) {
	raisedSomewhere := false
	for seed := int64(1); seed <= 5; seed++ {
		l, _ := heightsLevel(t, seed)
		if l.Floor(l.Spawn.X, l.Spawn.Y) != 0 {
			t.Fatalf("seed %d: spawn floor = %v", seed, l.Floor(l.Spawn.X, l.Spawn.Y))
		}
		if l.Floor(l.Exit.X, l.Exit.Y) > 0 {
			raisedSomewhere = true
		}
	}
	if !raisedSomewhere {
		t.Fatal("exit dais never raised across five seeds")
	}
}

func TestAssignHeightsCeilingsLeaveHeadroom(t *testing.T) {
	l, _ := heightsLevel(t, 4)
	for y := range l.H {
		for x := range l.W {
			if !l.At(x, y).Walkable() {
				continue
			}
			gap := l.Ceil(x, y) - l.Floor(x, y)
			if gap < level.DefaultMinHeadroom {
				t.Fatalf("no headroom at %d,%d: %v", x, y, gap)
			}
		}
	}
}

func TestAssignHeightsDeterministic(t *testing.T) {
	a, _ := heightsLevel(t, 7)
	b, _ := heightsLevel(t, 7)
	for i := range a.FloorH {
		if a.FloorH[i] != b.FloorH[i] || a.CeilH[i] != b.CeilH[i] {
			t.Fatal("same seed produced different heights")
		}
	}
}

func TestPlaceLiftLedge(t *testing.T) {
	// A corridor ending in a dead end: ledge at the end, lift on the neck.
	l := level.New(6, 3, 0)
	for x := 1; x <= 4; x++ {
		l.Carve(x, 1)
	}
	l.Spawn = geom.Coord{X: 1, Y: 1}

	// Chance 1 takes the first eligible dead end deterministically.
	ledge, ok := level.PlaceLiftLedge(l, worldgen.NewRNG(2), level.LiftConfig{Chance: 1}, nil)
	if !ok {
		t.Fatal("no ledge placed")
	}
	cfg := level.DefaultLiftConfig()
	want := float64(cfg.Raise) * cfg.Step
	if got := l.Floor(ledge.X, ledge.Y); got != want {
		t.Fatalf("ledge floor = %v, want %v", got, want)
	}
	if len(l.Lifts) != 1 {
		t.Fatalf("lifts = %d", len(l.Lifts))
	}
	for neck, lf := range l.Lifts {
		if lf.Low != 0 || lf.High != want {
			t.Fatalf("lift = %+v", lf)
		}
		if l.Ceil(neck.X, neck.Y) != want+1 || l.Ceil(ledge.X, ledge.Y) != want+1 {
			t.Fatal("shaft headroom not raised")
		}
	}
}

func TestPlaceLiftLedgeRespectsAvoid(t *testing.T) {
	l := level.New(6, 3, 0)
	for x := 1; x <= 4; x++ {
		l.Carve(x, 1)
	}
	avoidAll := func(geom.Coord) bool { return true }
	if _, ok := level.PlaceLiftLedge(l, worldgen.NewRNG(2), level.LiftConfig{Chance: 1}, avoidAll); ok {
		t.Fatal("avoid must veto every ledge")
	}
}

func TestLiftHeightCycle(t *testing.T) {
	lf := level.Lift{Low: 0, High: 1}
	const dwell, travel = 2.0, 1.5
	cases := []struct {
		t    float64
		want float64
	}{
		{0, 0},         // resting low
		{1.9, 0},       // still low
		{2.75, 0.5},    // halfway up
		{3.5, 1},       // arrived high
		{5.4, 1},       // resting high
		{6.25, 0.5},    // halfway down
		{7 + 2*3.5, 0}, // next cycle, resting low
	}
	for _, c := range cases {
		if got := level.LiftHeight(lf, c.t, dwell, travel); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("LiftHeight(t=%v) = %v, want %v", c.t, got, c.want)
		}
	}
}
