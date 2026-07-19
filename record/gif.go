// Package record captures frames from a running visualizer into demo GIFs
// and PNG screenshots, the way the family's --record flags and screenshot
// keys do.
//
// A [Recorder] (from [NewRecorder]) accumulates downscaled, dithered
// frames via [Recorder.Add] and writes a looping GIF with [Recorder.Save].
// For stills, [FromRGBA] wraps a software renderer's raw framebuffer as an
// image and [SavePNG] writes it out.
package record

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
)

// Recorder accumulates downscaled, dithered frames for a demo GIF.
type Recorder struct {
	frames    []*image.Paletted
	delayCs   int
	scale     int
	maxFrames int
	done      bool
}

// NewRecorder returns a recorder that captures at the given frames per
// second, downscaling each frame by scale. maxFrames caps the recording;
// zero means unlimited. Out-of-range arguments are clamped to sane values.
func NewRecorder(fps, scale, maxFrames int) *Recorder {
	return &Recorder{
		delayCs:   max(100/min(max(fps, 1), 100), 1),
		scale:     max(scale, 1),
		maxFrames: max(maxFrames, 0),
	}
}

// Add captures one frame. Frames past the cap are dropped.
func (r *Recorder) Add(img image.Image) {
	if r.done {
		return
	}
	small := downscale(img, r.scale)
	p := image.NewPaletted(small.Bounds(), palette.Plan9)
	draw.FloydSteinberg.Draw(p, small.Bounds(), small, image.Point{})
	r.frames = append(r.frames, p)
	if r.maxFrames > 0 && len(r.frames) >= r.maxFrames {
		r.done = true
	}
}

// Len returns the number of captured frames.
func (r *Recorder) Len() int { return len(r.frames) }

// Done reports whether the recorder has reached its frame cap.
func (r *Recorder) Done() bool { return r.done }

// Save writes the captured frames as a looping GIF at path. It fails when
// nothing was captured.
func (r *Recorder) Save(path string) error {
	if len(r.frames) == 0 {
		return fmt.Errorf("record: no frames captured")
	}
	delays := make([]int, len(r.frames))
	for i := range delays {
		delays[i] = r.delayCs
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("record: create %q: %w", path, err)
	}
	if err := gif.EncodeAll(f, &gif.GIF{Image: r.frames, Delay: delays}); err != nil {
		_ = f.Close()
		return fmt.Errorf("record: encode %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("record: close %q: %w", path, err)
	}
	return nil
}

func downscale(img image.Image, factor int) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx()/factor, b.Dy()/factor
	out := image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	for y := range out.Bounds().Dy() {
		for x := range out.Bounds().Dx() {
			out.Set(x, y, img.At(b.Min.X+x*factor, b.Min.Y+y*factor))
		}
	}
	return out
}
