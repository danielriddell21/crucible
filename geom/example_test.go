package geom_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/geom"
)

// Example steers an agent one normalized step toward a target, the way the
// family's sims move things around.
func Example() {
	pos := geom.Vec2{X: 1, Y: 1}
	target := geom.Vec2{X: 4, Y: 5}
	step := target.Sub(pos).Normalize()
	pos = pos.Add(step)
	fmt.Printf("moved to (%.1f, %.1f), %.0f left\n", pos.X, pos.Y, pos.Dist(target))
	// Output: moved to (1.6, 1.8), 4 left
}

// ExampleVec2_ToroidalDist shows distance on a wrapping world: crossing the
// seam is shorter than walking across the map.
func ExampleVec2_ToroidalDist() {
	a := geom.Vec2{X: 1, Y: 5}
	b := geom.Vec2{X: 9, Y: 5}
	fmt.Println(a.Dist(b), a.ToroidalDist(b, 10, 10))
	// Output: 8 2
}

func ExampleRect_Center() {
	room := geom.Rect{X: 2, Y: 4, W: 6, H: 8}
	c := room.Center()
	fmt.Println(c, room.Contains(c))
	// Output: {5 8} true
}

func ExampleClamp() {
	fmt.Println(geom.Clamp(15, 0, 10), geom.Clamp(-0.5, 0.0, 1.0), geom.Clamp("m", "a", "f"))
	// Output: 10 0 f
}
