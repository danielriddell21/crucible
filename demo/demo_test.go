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
