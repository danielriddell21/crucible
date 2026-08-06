// Package record captures frames from a running visualizer into demo GIFs,
// MP4 clips and PNG screenshots, the way the family's --record flags and
// screenshot keys do.
//
// A [Recorder] (from [NewRecorder]) accumulates downscaled, dithered
// frames via [Recorder.Add] and writes a looping GIF with [Recorder.Save].
// [WithVideo] — applied automatically by [New] for an .mp4 path — switches it
// to an H.264 MP4 instead, which stays small where a long GIF would not.
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

// Recorder accumulates downscaled frames for a demo GIF, or — in video mode,
// see [WithVideo] — raw frames for an MP4.
type Recorder struct {
	frames    []*image.Paletted
	disposal  []byte
	prev      *image.Paletted
	pal       color.Palette
	delayCs   int
	finalCs   int
	scale     int
	maxFrames int
	diff      bool
	done      bool

	// Video mode keeps the frames as raw RGBA instead of quantising them, so
	// ffmpeg encodes true colour rather than a palette's approximation.
	video      bool
	raw        [][]byte
	rawW, rawH int
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

// WithFrameDiff stores each frame as only the rectangle that changed since
// the previous one, keeping the earlier pixels via GIF frame disposal. For a
// mostly-static scene — a dashboard, a map beside a panel — this shrinks the
// file dramatically. It quantises without dithering so unchanged regions stay
// byte-identical between frames; pair it with [WithPalette] when the scene's
// colours are known.
func WithFrameDiff() Option {
	return func(r *Recorder) { r.diff = true }
}

// WithVideo makes the recorder keep raw RGBA frames and write an H.264 MP4
// from [Recorder.Save] instead of a GIF. Video stays small where a long GIF
// would not, and skips palette quantisation entirely, so the palette and
// frame-diff options no longer apply. It needs ffmpeg on PATH; without it Save
// returns [ErrNoFFmpeg]. [New] applies this automatically when the recording
// path ends in .mp4.
func WithVideo() Option {
	return func(r *Recorder) { r.video = true }
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
	if r.video {
		r.addRaw(img)
		return
	}
	small := downscale(img, r.scale)
	full := image.NewPaletted(small.Bounds(), r.pal)
	if r.diff {
		// Nearest-colour so unchanged regions match the previous frame exactly.
		draw.Draw(full, small.Bounds(), small, image.Point{}, draw.Src)
	} else {
		draw.FloydSteinberg.Draw(full, small.Bounds(), small, image.Point{})
	}

	frame := full
	disposal := byte(gif.DisposalNone)
	if r.diff && r.prev != nil {
		box, changed := diffBox(r.prev, full)
		if !changed {
			box = image.Rect(0, 0, 1, 1) // GIF frames may not be empty
		}
		sub := image.NewPaletted(box, r.pal)
		draw.Draw(sub, box, full, box.Min, draw.Src)
		frame = sub
	}
	r.frames = append(r.frames, frame)
	r.disposal = append(r.disposal, disposal)
	if r.diff {
		r.prev = full
	}
	if r.maxFrames > 0 && len(r.frames) >= r.maxFrames {
		r.done = true
	}
}

// diffBox returns the smallest rectangle covering every pixel that differs
// between the two frames, and whether any pixel changed.
func diffBox(a, b *image.Paletted) (image.Rectangle, bool) {
	w, h := b.Rect.Dx(), b.Rect.Dy()
	minX, minY, maxX, maxY := w, h, -1, -1
	for y := range h {
		ra := a.Pix[y*a.Stride : y*a.Stride+w]
		rb := b.Pix[y*b.Stride : y*b.Stride+w]
		for x := range w {
			if ra[x] == rb[x] {
				continue
			}
			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
		}
	}
	if maxX < 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

// addRaw keeps a frame as raw RGBA for video encoding. ffmpeg is fed a fixed
// frame size, so frames that disagree with the first are ignored.
func (r *Recorder) addRaw(img image.Image) {
	small := downscale(img, r.scale)
	b := small.Bounds()
	if r.rawW == 0 {
		r.rawW, r.rawH = b.Dx(), b.Dy()
	}
	if b.Dx() != r.rawW || b.Dy() != r.rawH {
		return
	}
	r.raw = append(r.raw, append([]byte(nil), small.Pix...))
	if r.maxFrames > 0 && len(r.raw) >= r.maxFrames {
		r.done = true
	}
}

// Len returns the number of captured frames.
func (r *Recorder) Len() int {
	if r.video {
		return len(r.raw)
	}
	return len(r.frames)
}

// Done reports whether the recorder has reached its frame cap.
func (r *Recorder) Done() bool { return r.done }

// Save writes the captured frames to path: a looping GIF, or an H.264 MP4 when
// the recorder is in video mode ([WithVideo]). It fails when nothing was
// captured.
func (r *Recorder) Save(path string) error {
	if r.video {
		// The frame delay doubles as the video's frame rate, so GIF and MP4
		// clips built from the same recorder play at the same speed.
		return EncodeMP4(path, r.raw, r.rawW, r.rawH, 100/float64(r.delayCs))
	}
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
	if err := gif.EncodeAll(f, &gif.GIF{Image: r.frames, Delay: delays, Disposal: r.disposal}); err != nil {
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
