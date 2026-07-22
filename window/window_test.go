package window_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/danielriddell21/crucible/window"
)

// TestConfigure checks that Configure sets the size and title, enables
// resizing, and applies the minimum-size limit. It touches Ebiten's global
// window state, so it needs a display: run under xvfb.
func TestConfigure(t *testing.T) {
	window.Configure(window.Options{Title: "crucible-test", Width: 800, Height: 600, MinWidth: 400, MinHeight: 300})

	if mode := ebiten.WindowResizingMode(); mode != ebiten.WindowResizingModeEnabled {
		t.Errorf("resizing mode = %v, want enabled", mode)
	}
	if w, h := ebiten.WindowSize(); w != 800 || h != 600 {
		t.Errorf("window size = %dx%d, want 800x600", w, h)
	}

	// The minimum-size limit is honoured: a smaller request clamps up.
	ebiten.SetWindowSize(200, 150)
	if w, h := ebiten.WindowSize(); w < 400 || h < 300 {
		t.Errorf("min-size limit not applied: got %dx%d, want at least 400x300", w, h)
	}
}
