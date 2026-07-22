package paint_test

import (
	"fmt"
	"image/color"

	"github.com/danielriddell21/crucible/paint"
)

// Distance shading dims a wall's colour by a game-chosen factor.
func ExampleScale() {
	wall := color.RGBA{200, 120, 80, 255}
	fmt.Println(paint.Scale(wall, 0.5))
	// Output: {100 60 40 255}
}
