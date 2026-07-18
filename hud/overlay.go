// Package hud provides the timed status-line overlay used by the family's
// game front-ends: a single line of text posted for a number of frames, on
// either a diagnostic or a player-facing channel.
//
// An [Overlay] holds at most one live line. Game code calls [Overlay.Post]
// when something worth saying happens, the update loop calls [Overlay.Tick]
// once per frame, and the renderer asks [Overlay.Active] what to draw. The
// [Channel] ([Diagnostic] or [Notice]) lets the front-end route debug
// telemetry and player-facing lines differently.
package hud

import "sync"

// Channel classifies an overlay line.
type Channel uint8

const (
	// Diagnostic lines are debug telemetry, shown only when the front-end
	// has debug messages enabled.
	Diagnostic Channel = iota
	// Notice lines are player-facing and always shown.
	Notice
)

// Overlay holds at most one active line of text and counts down its
// remaining frames. It is safe for concurrent use.
type Overlay struct {
	mu        sync.Mutex
	text      string
	channel   Channel
	remaining int
}

// New returns an empty overlay.
func New() *Overlay {
	return &Overlay{}
}

// Post replaces the active line with text on the given channel for the given
// number of frames. An empty text or non-positive frame count clears the
// overlay.
func (o *Overlay) Post(text string, frames int, ch Channel) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if text == "" || frames <= 0 {
		o.text, o.channel, o.remaining = "", Diagnostic, 0
		return
	}
	o.text, o.channel, o.remaining = text, ch, frames
}

// Tick advances the countdown by one frame, clearing the line when it
// expires. Call it once per game update.
func (o *Overlay) Tick() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.remaining > 0 {
		o.remaining--
		if o.remaining == 0 {
			o.text = ""
		}
	}
}

// Active returns the current line, its channel, and whether it is still
// live.
func (o *Overlay) Active() (string, Channel, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.text, o.channel, o.remaining > 0
}
