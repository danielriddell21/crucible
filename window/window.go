// Package window centralises Ebiten window setup for the family's front-ends:
// initial size, title, and a single shared resizing policy. It is one of the
// few packages allowed to import Ebiten (alongside menu and camera), because
// configuring the display window is inherently a display concern.
package window

import "github.com/hajimehoshi/ebiten/v2"

// Options describes a front-end's window.
type Options struct {
	// Title is the window title.
	Title string
	// Width and Height are the initial window size in pixels.
	Width, Height int
	// MinWidth and MinHeight bound how small the window may be resized. Zero
	// leaves the corresponding limit unset.
	MinWidth, MinHeight int
}

// Configure applies o to the Ebiten window: it sets the initial size and title,
// enables resizing (the family's shared policy so every app behaves the same),
// and applies any minimum-size limits. Call it once before [ebiten.RunGame];
// apps that resize the window later may still call [ebiten.SetWindowSize].
func Configure(o Options) {
	ebiten.SetWindowSize(o.Width, o.Height)
	ebiten.SetWindowTitle(o.Title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if o.MinWidth > 0 || o.MinHeight > 0 {
		ebiten.SetWindowSizeLimits(o.MinWidth, o.MinHeight, -1, -1)
	}
}
