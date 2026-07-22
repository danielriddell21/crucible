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
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
)

// Recorder accumulates downscaled, dithered frames for a demo GIF.
type Recorder struct {
	frames    []*image.Paletted
	pal       color.Palette
	delayCs   int
	finalCs   int
	scale     int
	maxFrames int
	done      bool
}

// Option configures a [Recorder] at construction. See [WithPalette],
// [WithFrameDelay], and [WithFinalHold].
type Option func(*Recorder)

// WithPalette quantises frames to a caller-supplied palette instead of the
// default web-safe [palette.Plan9]. A palette tuned to the scene's own
// colours gives a cleaner GIF than the generic one. An empty palette is
// ignored.
func WithPalette(p color.Palette) Option {
	return func(r *Recorder) {
		if len(p) > 0 {
			r.pal = p
		}
	}
}

// WithFrameDelay overrides the per-frame delay (in centiseconds) that fps
// otherwise derives, for recordings paced one frame per event rather than by
// a steady frame rate. A non-positive value is ignored.
func WithFrameDelay(centis int) Option {
	return func(r *Recorder) {
		if centis > 0 {
			r.delayCs = centis
		}
	}
}

// WithFinalHold holds the last frame for the given number of centiseconds
// before the GIF loops, so a demo lingers on its final state. Zero (the
// default) keeps every frame at the uniform delay.
func WithFinalHold(centis int) Option {
	return func(r *Recorder) {
		if centis > 0 {
			r.finalCs = centis
		}
	}
}

// NewRecorder returns a recorder that captures at the given frames per
// second, downscaling each frame by scale. maxFrames caps the recording;
// zero means unlimited. Out-of-range arguments are clamped to sane values.
// Options tune the palette and timing.
func NewRecorder(fps, scale, maxFrames int, opts ...Option) *Recorder {
	r := &Recorder{
		pal:       palette.Plan9,
		delayCs:   max(100/min(max(fps, 1), 100), 1),
		scale:     max(scale, 1),
		maxFrames: max(maxFrames, 0),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Add captures one frame. Frames past the cap are dropped.
func (r *Recorder) Add(img image.Image) {
	if r.done {
		return
	}
	small := downscale(img, r.scale)
	p := image.NewPaletted(small.Bounds(), r.pal)
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
	if r.finalCs > 0 {
		delays[len(delays)-1] = r.finalCs
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
