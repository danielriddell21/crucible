package demo_test

import (
	"fmt"
	"image"
	"image/color"

	"github.com/danielriddell21/crucible/demo"
	"github.com/danielriddell21/crucible/record"
)

func ExampleClip() {
	// A stand-in for a game: a counter the "renderer" paints as a shade.
	tick := 0
	clip := demo.Clip{
		Frames: 4,
		Every:  3, // three simulation steps per captured frame
		Step:   func(int) error { tick++; return nil },
		Frame: func(int) image.Image {
			img := image.NewRGBA(image.Rect(0, 0, 2, 2))
			for i := 0; i < len(img.Pix); i += 4 {
				img.Pix[i], img.Pix[i+3] = uint8(tick*10), 255
			}
			return img
		},
	}
	rec := record.NewRecorder(12, 1, 0, record.WithFrameDiff())
	captured, err := clip.Record(rec)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d frames from %d steps\n", captured, tick)
	// rec.Save("docs/demos/clip.gif") writes the GIF.

	// Output:
	// 4 frames from 10 steps
}

func ExampleRamp() {
	// Two scene colours, each fanned into four brightness levels.
	pal := demo.Ramp([]color.RGBA{
		{R: 150, G: 110, B: 78, A: 255}, // wall
		{R: 44, G: 36, B: 30, A: 255},   // floor
	}, 4)
	fmt.Println(len(pal), "entries")
	// Pass it to the recorder: record.NewRecorder(0, 1, 0, record.WithPalette(pal))

	// Output:
	// 9 entries
}
