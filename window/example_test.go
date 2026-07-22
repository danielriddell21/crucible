package window_test

import "github.com/danielriddell21/crucible/window"

func ExampleConfigure() {
	// A front-end configures its window once, then hands control to Ebiten.
	window.Configure(window.Options{
		Title:     "galapagos",
		Width:     1024,
		Height:    768,
		MinWidth:  512,
		MinHeight: 384,
	})
	// ebiten.RunGame(game) follows.
}
