package level_test

import (
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/level"
	"github.com/danielriddell21/crucible/worldgen"
)

// ventLevel digs a dungeon and carves vents into it.
func ventLevel(t *testing.T, seed int64, cfg level.VentConfig) (*level.Level, []geom.Rect) {
	t.Helper()
	l, rooms := generate(t, seed)
	level.CarveVents(l, rooms, cfg)
	return l, rooms
}

func ventCells(l *level.Level) []geom.Coord {
	var out []geom.Coord
	for y := range l.H {
		for x := range l.W {
			if l.At(x, y) == level.TileVent {
				out = append(out, geom.Coord{X: x, Y: y})
			}
		}
	}
	return out
}

func TestCarveVentsProducesConnectedNetworks(t *testing.T) {
	l, roomRects := ventLevel(t, 1, level.VentConfig{})
	vents := ventCells(l)
	if len(vents) == 0 {
		t.Fatal("no vents carved")
	}
	if len(l.VentMouths) < 2 {
		t.Fatalf("vent mouths = %d", len(l.VentMouths))
	}

	// Every carved network must open into at least two distinct rooms.
	notVent := func(c geom.Coord) bool { return l.At(c.X, c.Y) != level.TileVent }
	seen := map[geom.Coord]bool{}
	for _, v := range vents {
		if seen[v] {
			continue
		}
		field := worldgen.FloodDist(l.W, l.H, v, notVent, nil)
		roomsReached := map[int]bool{}
		for _, o := range vents {
			if field.At(o) < 0 {
				continue
			}
			seen[o] = true
			for _, n := range worldgen.Neighbors4(o) {
				for ri, r := range roomRects {
					if r.Contains(n) && l.At(n.X, n.Y).Walkable() {
						roomsReached[ri] = true
					}
				}
			}
		}
		if len(roomsReached) < 2 {
			t.Fatalf("vent network at %v opens into %d rooms", v, len(roomsReached))
		}
	}
}

func TestCarveVentsMouthsOpenOntoGround(t *testing.T) {
	l, _ := ventLevel(t, 2, level.VentConfig{})
	for _, m := range l.VentMouths {
		if l.At(m.X, m.Y) != level.TileVent {
			t.Fatalf("mouth %v is not a vent cell", m)
		}
		grounded := false
		for _, n := range worldgen.Neighbors4(m) {
			if tt := l.At(n.X, n.Y); tt.Walkable() && tt != level.TileVent {
				grounded = true
			}
		}
		if !grounded {
			t.Fatalf("mouth %v opens onto nothing", m)
		}
	}
}

func TestCarveVentsDeterministic(t *testing.T) {
	a, _ := ventLevel(t, 3, level.VentConfig{})
	b, _ := ventLevel(t, 3, level.VentConfig{})
	for i := range a.Tiles {
		if a.Tiles[i] != b.Tiles[i] {
			t.Fatal("same seed produced different vents")
		}
	}
}

func TestCarveVentsDepthBiasHidesTunnels(t *testing.T) {
	// With a strong depth bias, tunnels should show themselves to the rooms
	// (cells adjacent to walkable ground) no more than the unbiased layout.
	exposed := func(cfg level.VentConfig) (exposed, total int) {
		l, _ := ventLevel(t, 4, cfg)
		for _, v := range ventCells(l) {
			total++
			for _, n := range worldgen.Neighbors4(v) {
				if tt := l.At(n.X, n.Y); tt.Walkable() && tt != level.TileVent {
					exposed++
					break
				}
			}
		}
		return exposed, total
	}
	unbiasedExposed, unbiasedTotal := exposed(level.VentConfig{MaxNetworks: 1, MaxMouths: 3, DepthBias: -1})
	biasedExposed, biasedTotal := exposed(level.VentConfig{MaxNetworks: 1, MaxMouths: 3, DepthBias: 6})
	if unbiasedTotal == 0 || biasedTotal == 0 {
		t.Skip("seed carved no vents to compare")
	}
	unbiasedRatio := float64(unbiasedExposed) / float64(unbiasedTotal)
	biasedRatio := float64(biasedExposed) / float64(biasedTotal)
	if biasedRatio > unbiasedRatio+1e-9 {
		t.Fatalf("depth bias made tunnels more exposed: %.2f > %.2f", biasedRatio, unbiasedRatio)
	}
}

func TestCarveVentsRespectsNetworkCap(t *testing.T) {
	l, _ := ventLevel(t, 5, level.VentConfig{MaxNetworks: 1})
	vents := ventCells(l)
	if len(vents) == 0 {
		t.Skip("seed carved no vents")
	}
	// All vent cells must belong to one connected network.
	notVent := func(c geom.Coord) bool { return l.At(c.X, c.Y) != level.TileVent }
	field := worldgen.FloodDist(l.W, l.H, vents[0], notVent, nil)
	for _, v := range vents {
		if field.At(v) < 0 {
			t.Fatalf("second network found at %v with MaxNetworks 1", v)
		}
	}
}
