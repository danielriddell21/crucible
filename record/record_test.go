package record_test

import (
	"flag"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"

	"github.com/danielriddell21/crucible/record"
)

func testFrame(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestRecorderCapturesAndSaves(t *testing.T) {
	r := record.NewRecorder(20, 2, 0)
	r.Add(testFrame(8, 6, color.RGBA{R: 200, A: 255}))
	r.Add(testFrame(8, 6, color.RGBA{B: 200, A: 255}))
	if r.Len() != 2 || r.Done() {
		t.Fatalf("Len = %d Done = %v", r.Len(), r.Done())
	}
	path := filepath.Join(t.TempDir(), "demo.gif")
	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Image) != 2 {
		t.Fatalf("frames = %d", len(g.Image))
	}
	// Downscale by 2: 8x6 -> 4x3.
	if b := g.Image[0].Bounds(); b.Dx() != 4 || b.Dy() != 3 {
		t.Fatalf("bounds = %v", b)
	}
	if g.Delay[0] != 5 { // 100/20 fps in centiseconds
		t.Fatalf("delay = %d", g.Delay[0])
	}
}

func TestRecorderFrameCap(t *testing.T) {
	r := record.NewRecorder(10, 1, 2)
	for range 5 {
		r.Add(testFrame(4, 4, color.RGBA{G: 100, A: 255}))
	}
	if r.Len() != 2 || !r.Done() {
		t.Fatalf("Len = %d Done = %v", r.Len(), r.Done())
	}
}

func TestRecorderOptions(t *testing.T) {
	// A palette paced one frame per event, holding the last frame longer.
	pal := color.Palette{color.RGBA{A: 255}, color.RGBA{R: 255, A: 255}}
	r := record.NewRecorder(20, 1, 0,
		record.WithPalette(pal),
		record.WithFrameDelay(70),
		record.WithFinalHold(400),
	)
	r.Add(testFrame(4, 4, color.RGBA{R: 255, A: 255}))
	r.Add(testFrame(4, 4, color.RGBA{R: 255, A: 255}))

	path := filepath.Join(t.TempDir(), "demo.gif")
	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if g.Delay[0] != 70 {
		t.Errorf("frame delay = %d, want 70 (WithFrameDelay)", g.Delay[0])
	}
	if last := g.Delay[len(g.Delay)-1]; last != 400 {
		t.Errorf("final delay = %d, want 400 (WithFinalHold)", last)
	}
	// The pure-red frame must quantise to the red entry of the custom palette.
	rr, gg, bb, _ := g.Image[0].At(0, 0).RGBA()
	if rr>>8 != 255 || gg != 0 || bb != 0 {
		t.Errorf("pixel = (%d,%d,%d), want red from the custom palette", rr>>8, gg>>8, bb>>8)
	}
}

func TestEmptyPaletteIgnored(t *testing.T) {
	r := record.NewRecorder(10, 1, 0, record.WithPalette(nil))
	r.Add(testFrame(2, 2, color.RGBA{G: 200, A: 255}))
	if r.Len() != 1 {
		t.Fatalf("Len = %d; empty palette should fall back to the default", r.Len())
	}
}

func TestFrameDiffShrinksStaticScene(t *testing.T) {
	r := record.NewRecorder(10, 1, 0, record.WithFrameDiff())
	r.Add(testFrame(20, 20, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	// Second frame differs only in a 3x3 corner.
	f2 := testFrame(20, 20, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	for y := range 3 {
		for x := range 3 {
			f2.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	r.Add(f2)

	path := filepath.Join(t.TempDir(), "diff.gif")
	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Image) != 2 {
		t.Fatalf("frames = %d, want 2", len(g.Image))
	}
	if b := g.Image[0].Bounds(); b.Dx() != 20 || b.Dy() != 20 {
		t.Errorf("first frame = %v, want full 20x20", b)
	}
	if b := g.Image[1].Bounds(); b.Dx() >= 20 && b.Dy() >= 20 {
		t.Errorf("diff frame = %v, want a shrunk sub-rectangle", b)
	}
}

func TestOptionsAddFlagsAndDefaults(t *testing.T) {
	// A pre-set field becomes that flag's default; zero fields use canonical ones.
	o := record.Options{Scale: 2}
	fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
	o.AddFlags(fs)
	if err := fs.Parse([]string{"--record", "out.gif", "--record-frames", "120"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !o.Recording() || o.Path != "out.gif" {
		t.Errorf("Path = %q, Recording = %v", o.Path, o.Recording())
	}
	if o.FPS != 30 { // canonical default
		t.Errorf("FPS = %d, want 30", o.FPS)
	}
	if o.Scale != 2 { // preset default preserved
		t.Errorf("Scale = %d, want 2", o.Scale)
	}
	if o.Frames != 120 { // overridden by flag
		t.Errorf("Frames = %d, want 120", o.Frames)
	}
}

func TestOptionsAddPacedFlags(t *testing.T) {
	// A paced recorder exposes only --record and --record-frames; --record-fps
	// and --record-scale are deliberately absent.
	o := record.Options{}
	fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
	o.AddPacedFlags(fs)
	if fs.Lookup("record-fps") != nil || fs.Lookup("record-scale") != nil {
		t.Error("paced flags must not register --record-fps or --record-scale")
	}
	if err := fs.Parse([]string{"--record", "clip.gif", "--record-frames", "40"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !o.Recording() || o.Path != "clip.gif" || o.Frames != 40 {
		t.Errorf("Path = %q, Frames = %d, Recording = %v", o.Path, o.Frames, o.Recording())
	}
}

func TestOptionsAddPacedFlagsFramesDefault(t *testing.T) {
	// A pre-set Frames becomes the flag's default; the zero value stays zero
	// (record the whole run).
	o := record.Options{Frames: 150}
	fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
	o.AddPacedFlags(fs)
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if o.Frames != 150 {
		t.Errorf("Frames = %d, want preset default 150", o.Frames)
	}
}

func TestNewFromOptions(t *testing.T) {
	r := record.New(record.Options{FPS: 20, Scale: 2, Frames: 2})
	r.Add(testFrame(8, 8, color.RGBA{R: 200, A: 255}))
	r.Add(testFrame(8, 8, color.RGBA{G: 200, A: 255}))
	if !r.Done() { // capped at 2 frames
		t.Errorf("recorder should be done at its frame cap")
	}
	if !(record.Options{Path: "x"}).Recording() || (record.Options{}).Recording() {
		t.Error("Recording() should track whether a path is set")
	}
}

func TestSaveEmptyFails(t *testing.T) {
	r := record.NewRecorder(10, 1, 0)
	if err := r.Save(filepath.Join(t.TempDir(), "empty.gif")); err == nil {
		t.Fatal("Save of empty recording must fail")
	}
}

func TestRecorderClampsArguments(t *testing.T) {
	r := record.NewRecorder(0, 0, -1) // nonsense in, sane defaults out
	r.Add(testFrame(2, 2, color.RGBA{A: 255}))
	if r.Len() != 1 || r.Done() {
		t.Fatalf("Len = %d Done = %v", r.Len(), r.Done())
	}
}

func TestSavePNGRoundTrip(t *testing.T) {
	fb := make([]byte, 4*3*4)
	for i := 0; i < len(fb); i += 4 {
		fb[i], fb[i+3] = 255, 255 // opaque red
	}
	img := record.FromRGBA(fb, 4, 3)
	path := filepath.Join(t.TempDir(), "shot.png")
	if err := record.SavePNG(path, img); err != nil {
		t.Fatalf("SavePNG: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b := decoded.Bounds(); b.Dx() != 4 || b.Dy() != 3 {
		t.Fatalf("bounds = %v", b)
	}
	r, _, _, _ := decoded.At(1, 1).RGBA()
	if r != 0xffff {
		t.Fatalf("pixel red = %#x", r)
	}
}

func TestOptionsAddStdFlags(t *testing.T) {
	// The stdlib registration must produce the same flag names, defaults and
	// behaviour as the pflag one, so a front-end built on flag answers to the
	// same --record contract as one built on cobra.
	o := record.Options{Scale: 2}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	o.AddStdFlags(fs)
	if err := fs.Parse([]string{"-record", "out.gif", "-record-frames", "120"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !o.Recording() || o.Path != "out.gif" {
		t.Errorf("Path = %q, Recording = %v", o.Path, o.Recording())
	}
	if o.FPS != 30 {
		t.Errorf("FPS = %d, want canonical default 30", o.FPS)
	}
	if o.Scale != 2 {
		t.Errorf("Scale = %d, want preset default 2", o.Scale)
	}
	if o.Frames != 120 {
		t.Errorf("Frames = %d, want 120", o.Frames)
	}
}

func TestOptionsAddStdFlagsMatchesPflagNames(t *testing.T) {
	var std, pf record.Options
	sfs := flag.NewFlagSet("t", flag.ContinueOnError)
	std.AddStdFlags(sfs)
	pfs := pflag.NewFlagSet("t", pflag.ContinueOnError)
	pf.AddFlags(pfs)

	for _, name := range []string{"record", "record-fps", "record-scale", "record-frames"} {
		s, p := sfs.Lookup(name), pfs.Lookup(name)
		if s == nil {
			t.Errorf("stdlib set is missing -%s", name)
			continue
		}
		if p == nil {
			t.Errorf("pflag set is missing --%s", name)
			continue
		}
		if s.Usage != p.Usage {
			t.Errorf("%s usage differs:\n stdlib %q\n pflag  %q", name, s.Usage, p.Usage)
		}
		if s.DefValue != p.DefValue {
			t.Errorf("%s default differs: stdlib %q, pflag %q", name, s.DefValue, p.DefValue)
		}
	}
}

func TestOptionsAddPacedStdFlags(t *testing.T) {
	o := record.Options{}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	o.AddPacedStdFlags(fs)
	if fs.Lookup("record-fps") != nil || fs.Lookup("record-scale") != nil {
		t.Error("paced flags must not register -record-fps or -record-scale")
	}
	if err := fs.Parse([]string{"-record", "clip.gif", "-record-frames", "40"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !o.Recording() || o.Path != "clip.gif" || o.Frames != 40 {
		t.Errorf("Path = %q, Frames = %d, Recording = %v", o.Path, o.Frames, o.Recording())
	}
}
