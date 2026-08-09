// Package menu provides the keyboard-driven menu model shared by the
// family's title, pause, and settings screens: a titled list of items with
// wraparound selection, adjustable settings rows, and a software-canvas
// renderer.
//
// A [Menu] is a titled list of [Item] values; plain items carry an Action,
// settings rows carry an Adjust. Each frame the game feeds [Menu.Update] an
// [Input] snapshot and plays the returned [Sound]. [Menu.Draw] renders onto a
// [canvas.Canvas] with a [Theme] ([DefaultTheme] is the family's dark
// palette), and [Bar], [OnOff], and [Number] format settings-row values.
//
// This package is display-free: it takes an [Input] and produces pixels in a
// software canvas, and links no graphics library at all. An Ebiten front-end
// fills the Input with
// [github.com/danielriddell21/crucible/menu/ebiteninput.Poll], which reads the
// family's conventional bindings; a front-end on any other backend fills it
// from its own key state and gets the same menus.
package menu

import (
	"fmt"
	"image/color"

	"github.com/danielriddell21/crucible/canvas"
)

// Item is one menu row. Label is re-evaluated every frame so rows can show
// live values. Action runs on select; Adjust, when set, marks the row as an
// adjustable setting and runs on left/right with direction -1 or +1.
type Item struct {
	Label  func() string
	Action func()
	Adjust func(dir int)
}

// Menu is a titled list of items with a current selection.
type Menu struct {
	Title    string
	Subtitle []string
	Items    []Item
	Sel      int
}

// Sound reports which feedback cue an update produced, so the front-end can
// play its own audio.
type Sound int

const (
	// SoundNone means the update changed nothing.
	SoundNone Sound = iota
	// SoundMove means the selection moved or a setting was adjusted.
	SoundMove
	// SoundSelect means an item's action ran.
	SoundSelect
)

// Input is a one-frame snapshot of menu navigation intent. Each field is
// true only on the frame the key was first pressed.
type Input struct {
	Up, Down    bool
	Left, Right bool
	Select      bool
}

// Move moves the selection by dir rows, wrapping at either end.
func (m *Menu) Move(dir int) {
	m.Sel = (m.Sel + dir + len(m.Items)) % len(m.Items)
}

// Update applies one frame of input to the menu and reports the feedback
// sound to play.
func (m *Menu) Update(in Input) Sound {
	if len(m.Items) == 0 {
		return SoundNone
	}
	if in.Down {
		m.Move(1)
		return SoundMove
	}
	if in.Up {
		m.Move(-1)
		return SoundMove
	}
	cur := m.Items[m.Sel]
	if cur.Adjust != nil {
		if in.Left {
			cur.Adjust(-1)
			return SoundMove
		}
		if in.Right {
			cur.Adjust(1)
			return SoundMove
		}
	}
	if in.Select && cur.Action != nil {
		cur.Action()
		return SoundSelect
	}
	return SoundNone
}

// Theme is the colour scheme a menu draws with.
type Theme struct {
	Title      color.RGBA
	Text       color.RGBA
	Dim        color.RGBA
	Selected   color.RGBA
	SelectedBG color.RGBA
}

// DefaultTheme returns the family's conventional dark menu palette.
func DefaultTheme() Theme {
	return Theme{
		Title:      color.RGBA{R: 210, G: 70, B: 60, A: 255},
		Text:       color.RGBA{R: 200, G: 214, B: 208, A: 255},
		Dim:        color.RGBA{R: 120, G: 134, B: 128, A: 255},
		Selected:   color.RGBA{R: 90, G: 220, B: 160, A: 255},
		SelectedBG: color.RGBA{R: 24, G: 40, B: 34, A: 255},
	}
}

// Draw renders the menu onto the canvas: title, subtitle lines, then the
// items with the selection highlighted.
func (m *Menu) Draw(c *canvas.Canvas, th Theme) {
	w, _ := c.Size()
	c.TextCentered(70, m.Title, th.Title)
	y := 108
	for _, line := range m.Subtitle {
		c.TextCentered(y, line, th.Dim)
		y += 15
	}
	y = max(y+16, 176)
	for i, it := range m.Items {
		col := th.Text
		label := it.Label()
		if i == m.Sel {
			col = th.Selected
			label = "> " + label + " <"
			c.Rect(0, y-13, w, 18, th.SelectedBG)
		}
		c.TextCentered(y, label, col)
		y += 22
	}
}

// OnOff renders a boolean as the conventional "ON"/"OFF" setting value.
func OnOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// Bar renders a value in [0, 1] as a ten-cell meter for settings rows.
func Bar(v float64) string {
	filled := int(v*10 + 0.5)
	out := make([]byte, 10)
	for i := range out {
		if i < filled {
			out[i] = '#'
		} else {
			out[i] = '-'
		}
	}
	return string(out)
}

// Number renders a float setting value with one decimal place.
func Number(v float64) string { return fmt.Sprintf("%.1f", v) }
