package paint_test

import (
	"image/color"
	"testing"

	"github.com/danielriddell21/crucible/paint"
)

func TestScaleHalvesBrightness(t *testing.T) {
	got := paint.Scale(color.RGBA{100, 200, 50, 123}, 0.5)
	want := color.RGBA{50, 100, 25, 255}
	if got != want {
		t.Errorf("Scale = %v, want %v (opaque, RGB halved)", got, want)
	}
}

func TestScaleBytesMatchesScale(t *testing.T) {
	c := color.RGBA{33, 77, 199, 10}
	s := paint.Scale(c, 0.3)
	b := paint.ScaleBytes(c, 0.3)
	if b != [4]byte{s.R, s.G, s.B, s.A} {
		t.Errorf("ScaleBytes = %v, want %v", b, [4]byte{s.R, s.G, s.B, s.A})
	}
}

func TestBlendOverFullAlphaReplaces(t *testing.T) {
	fb := []byte{0, 0, 0, 255, 10, 20, 30, 255}
	paint.BlendOver(fb, color.RGBA{80, 90, 100, 255}, 1)
	want := []byte{80, 90, 100, 255, 80, 90, 100, 255}
	for i, v := range want {
		if fb[i] != v {
			t.Fatalf("BlendOver full alpha = %v, want %v", fb, want)
		}
	}
}

func TestBlendOverHalf(t *testing.T) {
	fb := []byte{100, 100, 100, 255}
	paint.BlendOver(fb, color.RGBA{200, 200, 200, 255}, 0.5)
	// 100*0.5 + 200*0.5 = 150
	if fb[0] != 150 || fb[1] != 150 || fb[2] != 150 {
		t.Errorf("BlendOver half = %v, want 150s", fb[:3])
	}
	if fb[3] != 255 {
		t.Errorf("alpha byte changed to %d, want 255 untouched", fb[3])
	}
}

func TestBlendOverIgnoresTrailingPartialPixel(t *testing.T) {
	fb := []byte{0, 0, 0, 255, 9, 9} // 1.5 pixels
	paint.BlendOver(fb, color.RGBA{255, 255, 255, 255}, 1)
	if fb[4] != 9 || fb[5] != 9 {
		t.Errorf("trailing partial pixel was modified: %v", fb)
	}
}
