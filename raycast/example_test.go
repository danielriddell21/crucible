package raycast_test

import (
	"fmt"
	"math"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/raycast"
)

// ExampleCast walks a ray from the player to the wall it hits, the core of
// a raycasting frame's per-column loop.
func ExampleCast() {
	// A 5x5 room whose border cells are solid.
	solid := func(x, y int) bool { return x <= 0 || y <= 0 || x >= 4 || y >= 4 }

	cam := raycast.NewCamera(geom.Vec2{X: 2.5, Y: 2.5}, 0, math.Pi/2)
	hit := raycast.Cast(cam.Pos, 1, 0, solid) // straight ahead, +X

	fmt.Printf("wall %v at %.1f\n", hit.Cell, hit.Dist)
	// Output: wall {4 2} at 1.5
}

// ExampleCamera_Project places a sprite on screen and reports the column
// its centre lands on.
func ExampleCamera_Project() {
	cam := raycast.NewCamera(geom.Vec2{X: 0, Y: 0}, 0, math.Pi/2)
	p, ok := cam.Project(geom.Vec2{X: 4, Y: 0}, 320, 200, 1)
	fmt.Println(ok, p.ScreenX, p.Size)
	// Output: true 160 50
}
