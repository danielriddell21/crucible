package record_test

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/danielriddell21/crucible/record"
)

// ExampleRecorder captures three frames at 20 fps, half scale, and writes
// a demo GIF — what a game's --record flag does with each rendered frame.
func ExampleRecorder() {
	dir, _ := os.MkdirTemp("", "crucible-record-example")
	defer os.RemoveAll(dir)

	r := record.NewRecorder(20, 2, 0)
	for i := range 3 {
		frame := image.NewRGBA(image.Rect(0, 0, 64, 48))
		frame.Set(i, i, color.RGBA{R: 255, A: 255})
		r.Add(frame)
	}
	if err := r.Save(filepath.Join(dir, "demo.gif")); err != nil {
		fmt.Println("save:", err)
		return
	}
	fmt.Println(r.Len(), "frames")
	// Output: 3 frames
}

// ExampleSavePNG writes a software renderer's raw framebuffer as a
// screenshot.
func ExampleSavePNG() {
	dir, _ := os.MkdirTemp("", "crucible-png-example")
	defer os.RemoveAll(dir)

	const w, h = 320, 200
	fb := make([]byte, w*h*4) // the renderer's RGBA framebuffer
	err := record.SavePNG(filepath.Join(dir, "shot.png"), record.FromRGBA(fb, w, h))
	fmt.Println(err)
	// Output: <nil>
}
