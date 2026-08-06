package record_test

import (
	"errors"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/danielriddell21/crucible/record"
)

// haveFFmpeg reports whether the optional video encoder is installed; the MP4
// tests skip without it rather than failing a machine that only needs GIFs.
func haveFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func TestIsVideoPath(t *testing.T) {
	for path, want := range map[string]bool{
		"demo.mp4":          true,
		"docs/demos/x.MP4":  true,
		"demo.gif":          false,
		"demo":              false,
		"demo.mp4.gif":      false,
		"/tmp/a.b/clip.mp4": true,
	} {
		if got := record.IsVideoPath(path); got != want {
			t.Errorf("IsVideoPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestEncodeMP4NoFrames(t *testing.T) {
	err := record.EncodeMP4(filepath.Join(t.TempDir(), "empty.mp4"), nil, 8, 8, 25)
	if err == nil {
		t.Fatal("encoding zero frames must fail")
	}
}

func TestNewSelectsVideoModeFromPath(t *testing.T) {
	r := record.New(record.Options{Path: "clip.mp4", FPS: 25, Scale: 1, Frames: 2})
	r.Add(testFrame(8, 8, color.RGBA{R: 200, A: 255}))
	r.Add(testFrame(8, 8, color.RGBA{G: 200, A: 255}))
	if r.Len() != 2 {
		t.Errorf("Len = %d, want 2 raw frames", r.Len())
	}
	if !r.Done() {
		t.Error("video recorder should honour its frame cap")
	}
}

func TestNewKeepsGIFModeForGIFPath(t *testing.T) {
	r := record.New(record.Options{Path: "clip.gif", FPS: 25, Scale: 1, Frames: 1})
	r.Add(testFrame(8, 8, color.RGBA{B: 200, A: 255}))
	out := filepath.Join(t.TempDir(), "clip.gif")
	if err := r.Save(out); err != nil {
		t.Fatalf("save gif: %v", err)
	}
	if b, err := os.ReadFile(out); err != nil || len(b) < 6 || string(b[:3]) != "GIF" {
		t.Errorf("expected a GIF file, got %v (err %v)", len(b), err)
	}
}

func TestVideoRecorderSavesMP4(t *testing.T) {
	if testing.Short() {
		t.Skip("shelling out to ffmpeg is slow")
	}
	if !haveFFmpeg() {
		t.Skip("ffmpeg not installed")
	}
	r := record.NewRecorder(25, 1, 0, record.WithVideo())
	// A couple of differing frames, at an even size libx264 accepts.
	for i := range 4 {
		c := color.RGBA{R: uint8(40 * i), G: 80, A: 255}
		r.Add(testFrame(16, 16, c))
	}
	out := filepath.Join(t.TempDir(), "clip.mp4")
	if err := r.Save(out); err != nil {
		t.Fatalf("save mp4: %v", err)
	}
	fi, err := os.Stat(out)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("expected a non-empty mp4: size err %v", err)
	}
}

func TestEncodeMP4WithoutFFmpegReportsCause(t *testing.T) {
	if haveFFmpeg() {
		t.Skip("ffmpeg installed; cannot exercise the missing-binary path")
	}
	err := record.EncodeMP4(filepath.Join(t.TempDir(), "x.mp4"), [][]byte{make([]byte, 4*4*4)}, 4, 4, 25)
	if !errors.Is(err, record.ErrNoFFmpeg) {
		t.Errorf("err = %v, want ErrNoFFmpeg", err)
	}
}
