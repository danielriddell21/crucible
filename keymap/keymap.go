// Package keymap lays out an on-screen control-hint bar — the "key: action"
// row the family's apps show along the bottom of the window. It is display-free:
// it formats and positions the hints and returns them for the caller to draw
// with its own text face, wrapping to the window width using a caller-supplied
// width measurement so the layout is identical whatever font the app renders in.
package keymap

import "strings"

// sep joins adjacent bindings on a row.
const sep = " · "

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

// BottomBar lays bindings out as a control bar anchored to the bottom-left of a
// w×h screen. It formats each binding as "key: action", joins them with " · ",
// wraps to the width f allows, and stacks the wrapped rows so the last sits pad
// above the bottom edge. Rows are returned top-to-bottom; an empty bindings
// slice returns nil.
func BottomBar(bindings []Binding, w, h, pad int, f Face) []Line {
	rows := wrap(bindings, w-2*pad, f.Measure)
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
