package keymap_test

import (
	"reflect"
	"testing"

	"github.com/danielriddell21/crucible/keymap"
)

// oneByte measures a string as one pixel per byte — simple, deterministic
// metrics for exercising the wrapping arithmetic.
func oneByte(s string) int { return len(s) }

func TestBindingLabel(t *testing.T) {
	if got := (keymap.Binding{Key: "space", Action: "pause"}).Label(); got != "space: pause" {
		t.Errorf("Label = %q, want %q", got, "space: pause")
	}
}

func TestBottomBarEmpty(t *testing.T) {
	if lines := keymap.BottomBar(nil, 200, 100, 8, keymap.Face{LineHeight: 16, Measure: oneByte}); lines != nil {
		t.Errorf("empty bindings should return nil, got %v", lines)
	}
}

func TestBottomBarSingleLineWhenItFits(t *testing.T) {
	bindings := []keymap.Binding{{Key: "a", Action: "one"}, {Key: "b", Action: "two"}}
	lines := keymap.BottomBar(bindings, 1000, 100, 10, keymap.Face{LineHeight: 16, Measure: oneByte})
	if len(lines) != 1 {
		t.Fatalf("want 1 row, got %d: %v", len(lines), lines)
	}
	if lines[0].Text != "a: one · b: two" {
		t.Errorf("text = %q", lines[0].Text)
	}
	// Bottom-left origin: last (only) row sits pad + one line above the bottom.
	if lines[0].X != 10 || lines[0].Y != 100-10-16 {
		t.Errorf("origin = (%d,%d), want (10,%d)", lines[0].X, lines[0].Y, 100-10-16)
	}
}

func TestBottomBarWrapsAndStacks(t *testing.T) {
	bindings := []keymap.Binding{
		{Key: "a", Action: "one"}, // "a: one" = 6
		{Key: "b", Action: "two"}, // "b: two" = 6
		{Key: "c", Action: "three"},
	}
	// avail = w - 2*pad = 20 - 2*2 = 16. "a: one · b: two" is 15 (fits);
	// adding " · c: three" would overflow, so it wraps to a second row.
	lines := keymap.BottomBar(bindings, 20, 100, 2, keymap.Face{LineHeight: 10, Measure: oneByte})
	want := []keymap.Line{
		{X: 2, Y: 100 - 2 - 2*10, Text: "a: one · b: two"},
		{X: 2, Y: 100 - 2 - 1*10, Text: "c: three"},
	}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("layout mismatch:\n got %v\nwant %v", lines, want)
	}
}

func TestBottomBarKeepsOversizedBindingOnItsOwnRow(t *testing.T) {
	// A binding wider than the available width still gets a row of its own
	// rather than being dropped.
	bindings := []keymap.Binding{{Key: "verylongkey", Action: "does a lot"}}
	lines := keymap.BottomBar(bindings, 10, 40, 1, keymap.Face{LineHeight: 12, Measure: oneByte})
	if len(lines) != 1 || lines[0].Text != "verylongkey: does a lot" {
		t.Errorf("oversized binding row = %v", lines)
	}
}
