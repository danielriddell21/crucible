// Package ebiteninput reads the family's conventional menu key bindings from
// Ebiten.
//
// It is a package of its own so that [github.com/danielriddell21/crucible/menu]
// stays display-free. The menu model draws onto a software canvas and could
// always be driven by hand, but while the Ebiten polling lived beside it every
// importer linked Ebiten — which shut out the front-ends built on something
// else. A raylib game builds its own [menu.Input] and never sees this package;
// an Ebiten game imports it and gets the shared bindings for free.
package ebiteninput

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/danielriddell21/crucible/menu"
)

// Poll reads the family's conventional menu key bindings from Ebiten: arrows
// or WASD to navigate and adjust, enter or space to select. Call it once per
// Update and pass the result to [menu.Menu.Update].
func Poll() menu.Input {
	return menu.Input{
		Up:     inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW),
		Down:   inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS),
		Left:   inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA),
		Right:  inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyD),
		Select: inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace),
	}
}
