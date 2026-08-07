// Package keymap lays out an on-screen control-hint bar — the "key: action"
// row the family's apps show along the bottom of the window. It is display-free:
// it formats and positions the hints and returns them for the caller to draw
// with its own text face, wrapping to the window width using a caller-supplied
// width measurement so the layout is identical whatever font the app renders in.
package keymap

import "strings"

// sep joins adjacent bindings on a row. It stays inside ASCII: the family's
// software canvas draws in a 7x13 bitmap face that covers U+0020..U+007E and
// nothing else, so a prettier separator like "·" would come out as the
// missing-glyph box in every app that uses it.
const sep = " | "

// Binding is one control hint: a key or key combination and the action it
// triggers.
type Binding struct {
	// Key names the key or combination, e.g. "space", "+/-", "drag/arrows".
	Key string
	// Action describes what the key does, e.g. "pause", "new track".
	Action string
}

// Label renders the binding as "key: action".
func (b Binding) Label() string { return b.Key + ": " + b.Action }

// Face describes the caller's text metrics so the bar wraps and stacks
// correctly in whatever font the app draws with.
type Face struct {
	// LineHeight is the vertical distance between wrapped rows, in pixels.
	LineHeight int
	// Measure returns the pixel width of s in the caller's font. Wrapping uses
	// it so a row never exceeds the available width.
	Measure func(s string) int
}

// Line is one laid-out row of the bar: the text to draw and where to draw it.
// X and Y are the row's top-left origin, matching the position most text calls
// expect; add your font's ascent if yours draws from the baseline instead.
type Line struct {
	X, Y int
	Text string
}

// Rows formats bindings as "key: action", joins them with " | ", and wraps them
// into rows no wider than avail measured by measure. It is [BottomBar]'s layout
// without its placement, for hints that live somewhere the bottom-left corner
// is not — inside a stats panel, beside a simulation, as a menu subtitle. The
// caller positions the rows itself. An empty bindings slice returns nil.
func Rows(bindings []Binding, avail int, measure func(string) int) []string {
	return wrap(bindings, avail, measure)
}

// BottomBar lays bindings out as a control bar anchored to the bottom-left of a
// w×h screen. It formats each binding as "key: action", joins them with " | ",
// wraps to the width f allows, and stacks the wrapped rows so the last sits pad
// above the bottom edge. Rows are returned top-to-bottom; an empty bindings
// slice returns nil.
func BottomBar(bindings []Binding, w, h, pad int, f Face) []Line {
	rows := Rows(bindings, w-2*pad, f.Measure)
	if len(rows) == 0 {
		return nil
	}
	lines := make([]Line, len(rows))
	for i, r := range rows {
		lines[i] = Line{
			X:    pad,
			Y:    h - pad - (len(rows)-i)*f.LineHeight,
			Text: r,
		}
	}
	return lines
}

// CenterPrompt lays out a single binding as a contextual prompt, centred
// horizontally and sitting in the lower third of a w×h screen. Games use it for
// "look at a door → E: open" hints that appear only while an interactive object
// is in focus. The returned Line shares the top-left origin convention of
// [BottomBar].
func CenterPrompt(b Binding, w, h int, f Face) Line {
	label := b.Label()
	return Line{
		X:    (w - f.Measure(label)) / 2,
		Y:    h*70/100 - f.LineHeight/2,
		Text: label,
	}
}

// wrap greedily packs binding labels into rows no wider than avail.
func wrap(bindings []Binding, avail int, measure func(string) int) []string {
	var rows []string
	var cur strings.Builder
	for _, b := range bindings {
		label := b.Label()
		if cur.Len() == 0 {
			cur.WriteString(label)
			continue
		}
		if measure(cur.String()+sep+label) > avail {
			rows = append(rows, cur.String())
			cur.Reset()
			cur.WriteString(label)
			continue
		}
		cur.WriteString(sep)
		cur.WriteString(label)
	}
	if cur.Len() > 0 {
		rows = append(rows, cur.String())
	}
	return rows
}
