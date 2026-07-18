package hud_test

import (
	"testing"

	"github.com/danielriddell21/crucible/hud"
)

func TestPostAndExpiry(t *testing.T) {
	o := hud.New()
	if _, _, ok := o.Active(); ok {
		t.Fatal("new overlay must be inactive")
	}
	o.Post("hello", 2, hud.Notice)
	text, ch, ok := o.Active()
	if !ok || text != "hello" || ch != hud.Notice {
		t.Fatalf("Active = %q %v %v", text, ch, ok)
	}
	o.Tick()
	if _, _, ok := o.Active(); !ok {
		t.Fatal("line must survive first tick")
	}
	o.Tick()
	if text, _, ok := o.Active(); ok || text != "" {
		t.Fatalf("line must expire after two ticks, got %q %v", text, ok)
	}
}

func TestPostReplaces(t *testing.T) {
	o := hud.New()
	o.Post("first", 10, hud.Diagnostic)
	o.Post("second", 5, hud.Notice)
	text, ch, ok := o.Active()
	if !ok || text != "second" || ch != hud.Notice {
		t.Fatalf("Active = %q %v %v", text, ch, ok)
	}
}

func TestEmptyPostClears(t *testing.T) {
	o := hud.New()
	o.Post("live", 10, hud.Notice)
	o.Post("", 10, hud.Notice)
	if _, _, ok := o.Active(); ok {
		t.Fatal("empty post must clear the overlay")
	}
	o.Post("live", 0, hud.Notice)
	if _, _, ok := o.Active(); ok {
		t.Fatal("zero-frame post must clear the overlay")
	}
}

func TestTickOnEmptyOverlay(t *testing.T) {
	o := hud.New()
	o.Tick() // must not panic or go negative
	if _, _, ok := o.Active(); ok {
		t.Fatal("ticked empty overlay must stay inactive")
	}
}
