package canvas_test

import (
	"fmt"
	"image/color"

	"github.com/danielriddell21/crucible/canvas"
)

// ExampleCanvas composes a small title screen the way the family's menus
// do: fill, highlight bar, centred text, then hand the pixels to Ebiten's
// WritePixels.
func ExampleCanvas() {
	c := canvas.New(320, 200)
	c.Fill(color.RGBA{R: 10, G: 12, B: 16, A: 255})
	c.Rect(0, 87, 320, 18, color.RGBA{R: 24, G: 40, B: 34, A: 255})
	c.TextCentered(100, "> DESCEND <", color.RGBA{R: 90, G: 220, B: 160, A: 255})

	w, h := c.Size()
	fmt.Println(w, h, len(c.Pixels()) == w*h*4)
	// Output: 320 200 true
}
