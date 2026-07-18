// Package status is the pipeline that turns game events into timed HUD
// lines: a game maps its telemetry into a cue type of its own, a [Source]
// decides whether the cue deserves a [Line] and with what wording, and
// [Emit] posts the line to a [hud.Overlay].
//
// The cue vocabulary belongs to each game; this package fixes only the
// contract between the pieces, so sources are interchangeable: a scripted
// table adapted with [Func], or the narrata-backed rewriter in package
// narrate.
package status

import "github.com/danielriddell21/crucible/hud"

// Line is one status message ready for the overlay.
type Line struct {
	Text    string
	Channel hud.Channel
	Frames  int
}

// Source turns a cue into zero or one lines. Implementations call emit for
// each line they produce and simply return to stay silent. C is the game's
// own cue type.
type Source[C any] interface {
	Request(cue C, emit func(Line))
}

// Func adapts a plain function to the Source interface.
type Func[C any] func(cue C, emit func(Line))

// Request implements Source by calling the function itself.
func (f Func[C]) Request(cue C, emit func(Line)) { f(cue, emit) }

// Emit returns an emit callback that posts lines to the overlay, for wiring
// a Source straight to a HUD.
func Emit(o *hud.Overlay) func(Line) {
	return func(l Line) { o.Post(l.Text, l.Frames, l.Channel) }
}
