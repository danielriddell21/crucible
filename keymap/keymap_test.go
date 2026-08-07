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
	if lines[0].Text != "a: one | b: two" {
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
	// avail = w - 2*pad = 20 - 2*2 = 16. "a: one | b: two" is 15 (fits);
	// adding " | c: three" would overflow, so it wraps to a second row.
	lines := keymap.BottomBar(bindings, 20, 100, 2, keymap.Face{LineHeight: 10, Measure: oneByte})
	want := []keymap.Line{
		{X: 2, Y: 100 - 2 - 2*10, Text: "a: one | b: two"},
		{X: 2, Y: 100 - 2 - 1*10, Text: "c: three"},
	}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("layout mismatch:\n got %v\nwant %v", lines, want)
	}
}

func TestCenterPrompt(t *testing.T) {
	// "E: open" is 7 bytes → 7px wide with oneByte; centred in a 100px screen
	// gives x = (100-7)/2 = 46, and y sits in the lower third.
	line := keymap.CenterPrompt(keymap.Binding{Key: "E", Action: "open"}, 100, 200, keymap.Face{LineHeight: 16, Measure: oneByte})
	if line.Text != "E: open" {
		t.Errorf("text = %q", line.Text)
	}
	if line.X != (100-7)/2 {
		t.Errorf("X = %d, want %d (centred)", line.X, (100-7)/2)
	}
	if line.Y != 200*70/100-8 {
		t.Errorf("Y = %d, want %d (lower third)", line.Y, 200*70/100-8)
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

func TestRowsWrapsWithoutPlacing(t *testing.T) {
	// The layout half of keymap.BottomBar, for hints an app positions itself.
	bindings := []keymap.Binding{
		{Key: "space", Action: "pause"},
		{Key: "r", Action: "restart"},
		{Key: "q", Action: "quit"},
	}
	measure := func(s string) int { return len(s) }

	if rows := keymap.Rows(bindings, 200, measure); len(rows) != 1 {
		t.Errorf("keymap.Rows in plenty of room = %d rows, want 1: %q", len(rows), rows)
	} else if rows[0] != "space: pause | r: restart | q: quit" {
		t.Errorf("row = %q", rows[0])
	}

	// Narrow enough that no two bindings share a row.
	const avail = 14
	rows := keymap.Rows(bindings, avail, measure)
	if len(rows) != len(bindings) {
		t.Fatalf("keymap.Rows in a narrow space = %d rows, want one per binding: %q", len(rows), rows)
	}
	for _, r := range rows {
		if len(r) > avail {
			t.Errorf("row %q is %d wide, over the %d allowed", r, len(r), avail)
		}
	}
}

func TestRowsEmpty(t *testing.T) {
	if rows := keymap.Rows(nil, 100, func(s string) int { return len(s) }); rows != nil {
		t.Errorf("keymap.Rows(nil) = %q, want nil", rows)
	}
}

func TestBottomBarUsesRows(t *testing.T) {
	// keymap.BottomBar is keymap.Rows plus placement; the text must match exactly.
	bindings := []keymap.Binding{{Key: "a", Action: "one"}, {Key: "b", Action: "two"}}
	f := keymap.Face{LineHeight: 10, Measure: func(s string) int { return len(s) }}
	lines := keymap.BottomBar(bindings, 40, 100, 4, f)
	rows := keymap.Rows(bindings, 40-2*4, f.Measure)
	if len(lines) != len(rows) {
		t.Fatalf("keymap.BottomBar gave %d lines, keymap.Rows gave %d", len(lines), len(rows))
	}
	for i := range lines {
		if lines[i].Text != rows[i] {
			t.Errorf("line %d: keymap.BottomBar %q, keymap.Rows %q", i, lines[i].Text, rows[i])
		}
	}
}
