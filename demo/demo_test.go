package demo_test

import (
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/danielriddell21/crucible/demo"
	"github.com/danielriddell21/crucible/record"
)

// solid returns a w×h image filled with c, standing in for a rendered frame.
func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, 255
	}
	return img
}

func TestClipCapturesRequestedFrames(t *testing.T) {
	steps := 0
	c := demo.Clip{
		Frames: 3,
		Step:   func(int) error { steps++; return nil },
		Frame:  func(int) image.Image { return solid(4, 4, color.RGBA{R: 200, A: 255}) },
	}
	rec := record.NewRecorder(25, 1, 0)
	got, err := c.Record(rec)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got != 3 || rec.Len() != 3 {
		t.Errorf("captured = %d, recorder len = %d, want 3 and 3", got, rec.Len())
	}
	if steps != 3 {
		t.Errorf("Step called %d times, want 3", steps)
	}
}

func TestClipEverySkipsIntermediateSteps(t *testing.T) {
	steps := 0
	c := demo.Clip{
		Frames: 3,
		Every:  4,
		Step:   func(int) error { steps++; return nil },
		Frame:  func(int) image.Image { return solid(4, 4, color.RGBA{G: 200, A: 255}) },
	}
	rec := record.NewRecorder(25, 1, 0)
	got, err := c.Record(rec)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got != 3 {
		t.Fatalf("captured = %d, want 3", got)
	}
	// Captures land on steps 0, 4 and 8, so the sim advanced nine times.
	if steps != 9 {
		t.Errorf("Step called %d times, want 9", steps)
	}
}

func TestClipReadyGatesCapture(t *testing.T) {
	c := demo.Clip{
		Frames:   2,
		MaxSteps: 50,
		Step:     func(int) error { return nil },
		Ready:    func(step int) bool { return step >= 10 },
		Frame:    func(int) image.Image { return solid(4, 4, color.RGBA{B: 200, A: 255}) },
	}
	rec := record.NewRecorder(25, 1, 0)
	got, err := c.Record(rec)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got != 2 {
		t.Errorf("captured = %d, want 2 once ready", got)
	}
}

func TestClipStopsAtMaxStepsWhenNeverReady(t *testing.T) {
	c := demo.Clip{
		Frames:   100,
		MaxSteps: 12,
		Step:     func(int) error { return nil },
		Ready:    func(int) bool { return false },
		Frame:    func(int) image.Image { return solid(4, 4, color.RGBA{A: 255}) },
	}
	rec := record.NewRecorder(25, 1, 0)
	got, err := c.Record(rec)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got != 0 {
		t.Errorf("captured = %d, want 0 when never ready", got)
	}
}

func TestClipStopsWhenTheRunFinishes(t *testing.T) {
	// A run of unknown length: it ends on its own after six steps.
	steps := 0
	c := demo.Clip{
		Frames:   100,
		MaxSteps: 100,
		Step:     func(int) error { steps++; return nil },
		Frame:    func(int) image.Image { return solid(4, 4, color.RGBA{A: 255}) },
		Stop:     func(step int) bool { return step >= 5 },
	}
	rec := record.NewRecorder(25, 1, 0)
	got, err := c.Record(rec)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	// Six frames: steps 0..5, with the finishing state captured before stopping.
	if got != 6 {
		t.Errorf("captured = %d, want 6 including the final state", got)
	}
	if steps != 6 {
		t.Errorf("Step called %d times, want 6", steps)
	}
}

func TestClipStopsAtRecorderCap(t *testing.T) {
	c := demo.Clip{
		Frames: 100,
		Step:   func(int) error { return nil },
		Frame:  func(int) image.Image { return solid(4, 4, color.RGBA{A: 255}) },
	}
	rec := record.NewRecorder(25, 1, 5) // recorder caps itself at five frames
	got, err := c.Record(rec)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got != 5 {
		t.Errorf("captured = %d, want 5 (the recorder's cap)", got)
	}
}

func TestClipPropagatesStepError(t *testing.T) {
	boom := errors.New("boom")
	c := demo.Clip{
		Frames: 10,
		Step:   func(step int) error { return boom },
		Frame:  func(int) image.Image { return solid(4, 4, color.RGBA{A: 255}) },
	}
	if _, err := c.Record(record.NewRecorder(25, 1, 0)); !errors.Is(err, boom) {
		t.Errorf("err = %v, want boom", err)
	}
}

func TestClipRejectsUnrunnableConfigs(t *testing.T) {
	rec := record.NewRecorder(25, 1, 0)
	frame := func(int) image.Image { return solid(2, 2, color.RGBA{A: 255}) }

	if _, err := (demo.Clip{Frames: 1, Frame: frame}).Record(rec); err == nil {
		t.Error("a clip without Step should fail")
	}
	if _, err := (demo.Clip{Frames: 1, Step: func(int) error { return nil }}).Record(rec); err == nil {
		t.Error("a clip without Frame should fail")
	}
	unbounded := demo.Clip{Step: func(int) error { return nil }, Frame: frame}
	if _, err := unbounded.Record(rec); err == nil {
		t.Error("a clip with neither Frames nor MaxSteps should fail rather than run forever")
	}
}

func TestMontageGridLayout(t *testing.T) {
	cells := []image.Image{
		solid(10, 6, color.RGBA{R: 255, A: 255}),
		solid(10, 6, color.RGBA{G: 255, A: 255}),
		solid(10, 6, color.RGBA{B: 255, A: 255}),
	}
	// Two columns, so three cells make two rows with the last one short.
	out := demo.Montage(cells, 2, 4, color.RGBA{A: 255})
	wantW, wantH := 2*10+3*4, 2*6+3*4
	if b := out.Bounds(); b.Dx() != wantW || b.Dy() != wantH {
		t.Fatalf("size = %dx%d, want %dx%d", b.Dx(), b.Dy(), wantW, wantH)
	}
	// The second cell sits one column across, past the gap.
	if got := out.RGBAAt(4+10+4, 4); got.G != 255 {
		t.Errorf("cell 2 top-left = %v, want green", got)
	}
	// The gap keeps the background colour.
	if got := out.RGBAAt(0, 0); got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("border = %v, want background", got)
	}
}

func TestMontageEmpty(t *testing.T) {
	if out := demo.Montage(nil, 3, 2, color.RGBA{A: 255}); out != nil {
		t.Errorf("no cells should give a nil montage, got %v", out.Bounds())
	}
}

func TestRamp(t *testing.T) {
	bases := []color.RGBA{{R: 200, G: 100, B: 50, A: 255}, {R: 10, G: 220, B: 30, A: 255}}
	const steps = 18
	pal := demo.Ramp(bases, steps)

	if len(pal) != 1+steps*len(bases) {
		t.Fatalf("len = %d, want %d", len(pal), 1+steps*len(bases))
	}
	if got := pal[0].(color.RGBA); got != (color.RGBA{A: 255}) {
		t.Errorf("pal[0] = %v, want opaque black", got)
	}
	// The ramp runs dim to full: the last shade of a base is the base itself.
	if got := pal[steps].(color.RGBA); got != bases[0] {
		t.Errorf("brightest shade = %v, want the base %v", got, bases[0])
	}
	// The dimmest shade keeps the hue rather than collapsing to black.
	if got := pal[1].(color.RGBA); got.R == 0 && got.G == 0 && got.B == 0 {
		t.Error("dimmest shade should retain the base hue")
	}
}

func TestRampDegenerate(t *testing.T) {
	if pal := demo.Ramp(nil, 8); len(pal) != 1 {
		t.Errorf("no bases should give just black, got %d entries", len(pal))
	}
	if pal := demo.Ramp([]color.RGBA{{R: 9, A: 255}}, 0); len(pal) != 1 {
		t.Errorf("zero steps should give just black, got %d entries", len(pal))
	}
}

func TestDownscaleAveragesBlocks(t *testing.T) {
	// A 4x4 image split into four 2x2 quadrants of one colour each. At factor
	// 2 every output pixel is the average of one uniform block, so the result
	// is those four colours exactly.
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	quad := [4]color.RGBA{
		{R: 200, A: 255},
		{G: 100, A: 255},
		{B: 40, A: 255},
		{R: 10, G: 20, B: 30, A: 255},
	}
	for y := range 4 {
		for x := range 4 {
			src.SetRGBA(x, y, quad[(y/2)*2+x/2])
		}
	}

	out := demo.Downscale(src, 2)
	if got := out.Bounds(); got.Dx() != 2 || got.Dy() != 2 {
		t.Fatalf("bounds = %v, want 2x2", got)
	}
	for y := range 2 {
		for x := range 2 {
			r, g, b, a := out.At(x, y).RGBA()
			want := quad[y*2+x]
			got := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			if got != want {
				t.Errorf("pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestDownscaleMixesWithinABlock(t *testing.T) {
	// Half black, half white down one 2x1 block averages to mid grey. This is
	// what separates Downscale from the recorder's point sampler, which would
	// return whichever pixel it happened to land on.
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{A: 255})
	src.SetRGBA(1, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	src.SetRGBA(0, 1, color.RGBA{A: 255})
	src.SetRGBA(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	r, _, _, _ := demo.Downscale(src, 2).At(0, 0).RGBA()
	if got := uint8(r >> 8); got != 127 {
		t.Errorf("red = %d, want 127", got)
	}
}

func TestDownscaleBelowTwoIsIdentity(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 3))
	for _, factor := range []int{0, 1, -4} {
		if got := demo.Downscale(src, factor); got != image.Image(src) {
			t.Errorf("Downscale(src, %d) returned a copy, want src itself", factor)
		}
	}
}

func TestDownscaleNeverReturnsAnEmptyImage(t *testing.T) {
	// A factor larger than the source would floor to zero pixels; the result
	// stays at least 1x1 so a caller can always draw it.
	src := image.NewRGBA(image.Rect(0, 0, 3, 3))
	got := demo.Downscale(src, 8).Bounds()
	if got.Dx() != 1 || got.Dy() != 1 {
		t.Errorf("bounds = %v, want 1x1", got)
	}
}
