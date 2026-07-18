package menu_test

import (
	"testing"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/menu"
)

func staticItem(label string, action func()) menu.Item {
	return menu.Item{Label: func() string { return label }, Action: action}
}

func TestMoveWraps(t *testing.T) {
	m := &menu.Menu{Items: []menu.Item{staticItem("a", nil), staticItem("b", nil), staticItem("c", nil)}}
	m.Move(-1)
	if m.Sel != 2 {
		t.Fatalf("Sel after wrap up = %d", m.Sel)
	}
	m.Move(1)
	if m.Sel != 0 {
		t.Fatalf("Sel after wrap down = %d", m.Sel)
	}
}

func TestUpdateNavigation(t *testing.T) {
	m := &menu.Menu{Items: []menu.Item{staticItem("a", nil), staticItem("b", nil)}}
	if s := m.Update(menu.Input{Down: true}); s != menu.SoundMove || m.Sel != 1 {
		t.Fatalf("Down: sound %v sel %d", s, m.Sel)
	}
	if s := m.Update(menu.Input{Up: true}); s != menu.SoundMove || m.Sel != 0 {
		t.Fatalf("Up: sound %v sel %d", s, m.Sel)
	}
	if s := m.Update(menu.Input{}); s != menu.SoundNone {
		t.Fatalf("idle: sound %v", s)
	}
}

func TestUpdateSelect(t *testing.T) {
	ran := false
	m := &menu.Menu{Items: []menu.Item{staticItem("go", func() { ran = true })}}
	if s := m.Update(menu.Input{Select: true}); s != menu.SoundSelect || !ran {
		t.Fatalf("Select: sound %v ran %v", s, ran)
	}
}

func TestUpdateAdjust(t *testing.T) {
	val := 0
	m := &menu.Menu{Items: []menu.Item{{
		Label:  func() string { return "setting" },
		Adjust: func(dir int) { val += dir },
	}}}
	if s := m.Update(menu.Input{Right: true}); s != menu.SoundMove || val != 1 {
		t.Fatalf("Right: sound %v val %d", s, val)
	}
	if s := m.Update(menu.Input{Left: true}); s != menu.SoundMove || val != 0 {
		t.Fatalf("Left: sound %v val %d", s, val)
	}
	// Select on a row without an action stays silent.
	if s := m.Update(menu.Input{Select: true}); s != menu.SoundNone {
		t.Fatalf("Select on adjustable: sound %v", s)
	}
}

func TestUpdateEmptyMenu(t *testing.T) {
	m := &menu.Menu{}
	if s := m.Update(menu.Input{Down: true}); s != menu.SoundNone {
		t.Fatalf("empty menu: sound %v", s)
	}
}

func TestDrawHighlightsSelection(t *testing.T) {
	m := &menu.Menu{
		Title:    "TITLE",
		Subtitle: []string{"a subtitle"},
		Items:    []menu.Item{staticItem("first", nil), staticItem("second", nil)},
	}
	c := canvas.New(320, 240)
	m.Draw(c, menu.DefaultTheme())
	// The selection background is a distinct colour; make sure it landed.
	th := menu.DefaultTheme()
	found := false
	p := c.Pixels()
	for i := 0; i < len(p); i += 4 {
		if p[i] == th.SelectedBG.R && p[i+1] == th.SelectedBG.G && p[i+2] == th.SelectedBG.B {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("selection background not drawn")
	}
}

func TestFormatters(t *testing.T) {
	if menu.OnOff(true) != "ON" || menu.OnOff(false) != "OFF" {
		t.Error("OnOff wrong")
	}
	if got := menu.Bar(0.5); got != "#####-----" {
		t.Errorf("Bar(0.5) = %q", got)
	}
	if got := menu.Bar(0); got != "----------" {
		t.Errorf("Bar(0) = %q", got)
	}
	if got := menu.Bar(1); got != "##########" {
		t.Errorf("Bar(1) = %q", got)
	}
	if got := menu.Number(1.25); got != "1.2" {
		t.Errorf("Number = %q", got)
	}
}
