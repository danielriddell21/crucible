package canvas_test

import (
	"image/color"
	"testing"

	"golang.org/x/image/font/basicfont"

	"github.com/danielriddell21/crucible/canvas"
)

func TestTextFaceDrawsAndAdvances(t *testing.T) {
	c := canvas.New(80, 20)
	adv := c.TextFace(2, 14, "AB", color.RGBA{R: 255, A: 255}, basicfont.Face7x13)
	if adv <= 0 {
		t.Errorf("advance = %d, want positive", adv)
	}
	painted := false
	for _, p := range c.Pixels() {
		if p != 0 {
			painted = true
			break
		}
	}
	if !painted {
		t.Error("TextFace drew nothing")
	}
}

func TestTextFaceNilFallsBackToBuiltIn(t *testing.T) {
	c := canvas.New(80, 20)
	adv := c.TextFace(2, 14, "hello", color.RGBA{G: 255, A: 255}, nil)
	if want := 5 * canvas.GlyphWidth; adv != want {
		t.Errorf("advance = %d, want %d from the built-in face", adv, want)
	}
}

func TestMeasureFace(t *testing.T) {
	if got, want := canvas.MeasureFace("abcd", nil), 4*canvas.GlyphWidth; got != want {
		t.Errorf("nil face measure = %d, want %d", got, want)
	}
	if got := canvas.MeasureFace("abcd", basicfont.Face7x13); got <= 0 {
		t.Errorf("measure = %d, want positive", got)
	}
	if canvas.MeasureFace("", basicfont.Face7x13) != 0 {
		t.Error("an empty string should measure zero")
	}
}
