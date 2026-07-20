package camera_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/camera"
)

// ExampleCamera zooms in under a screen point and pans, then maps a screen
// position back into the world — the loop a top-down visualizer runs from
// mouse input.
func ExampleCamera() {
	cam := camera.New()
	cam.ZoomAt(2, 0, 0, 100, 100) // magnify, anchored at the top-left
	cam.Pan(20, 0, 100, 100)      // slide right by 20 screen pixels

	wx, wy := cam.ScreenToWorld(0, 0)
	fmt.Printf("world (%.0f, %.0f) at zoom %.0f\n", wx, wy, cam.Zoom)
	// Output: world (10, 0) at zoom 2
}
