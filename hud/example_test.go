package hud_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/hud"
)

// ExampleOverlay posts a two-frame notice and ticks it to expiry, the way a
// game's update loop drives the HUD.
func ExampleOverlay() {
	o := hud.New()
	o.Post("A hidden place. Noted.", 2, hud.Notice)

	for range 3 {
		if text, ch, ok := o.Active(); ok {
			fmt.Println(text, ch == hud.Notice)
		} else {
			fmt.Println("(clear)")
		}
		o.Tick()
	}
	// Output:
	// A hidden place. Noted. true
	// A hidden place. Noted. true
	// (clear)
}
