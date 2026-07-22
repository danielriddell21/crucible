package keymap_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/keymap"
)

func ExampleBottomBar() {
	bindings := []keymap.Binding{
		{Key: "space", Action: "pause"},
		{Key: "+/-", Action: "speed"},
		{Key: "r", Action: "restart"},
	}
	// A fixed 6-pixel-per-character face, like the built-in debug font.
	face := keymap.Face{LineHeight: 16, Measure: func(s string) int { return len(s) * 6 }}
	for _, line := range keymap.BottomBar(bindings, 640, 360, 8, face) {
		fmt.Printf("(%d,%d) %s\n", line.X, line.Y, line.Text)
	}
	// Output:
	// (8,336) space: pause · +/-: speed · r: restart
}
