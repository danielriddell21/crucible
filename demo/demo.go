// Package demo assembles documentation media from a headless simulation: it
// drives a run and captures frames into a [record.Recorder] ([Clip]), shrinks
// a frame to a sensible size ([Downscale]), tiles stills into a contact sheet
// ([Montage]), and builds the brightness ramps a clean GIF palette needs
// ([Ramp]).
//
// It is display-free. A front-end that can draw a frame into a pixel buffer —
// its own software renderer, or a crucible canvas — generates its whole docs/demos set
// with no window and no display, which is how the family builds its media in
// CI.
//
// What each clip shows — which levels, which staging, which inputs — stays in
// the game. This package only drives and assembles.
package demo

import (
	"errors"
	"image"
	"image/color"
	"image/draw"

	"github.com/danielriddell21/crucible/record"
)

// Clip drives a simulation and captures frames from it. The callbacks carry
// all the app-specific behaviour: Step advances the game, Frame renders it,
// and Ready decides when the interesting part has started.
type Clip struct {
	// Frames is how many frames to capture before stopping. Zero captures
	// until MaxSteps is reached.
	Frames int
	// Every advances the simulation this many steps per captured frame, so a
	// long run can be shown at a watchable pace. Zero and one both mean every
	// step is captured.
	Every int
	// MaxSteps caps total simulation steps so a clip whose Ready never fires
	// still terminates. Zero means no cap, in which case Frames must be set.
	MaxSteps int

	// Step advances the simulation by one step. It is called for every step,
	// captured or not.
	Step func(step int) error
	// Ready gates capture: nothing is recorded until it first returns true, so
	// a clip can open on the action rather than the walk up to it. Nil
	// captures from the first step.
	Ready func(step int) bool
	// Frame renders the current state. It is called only for captured frames.
	Frame func(step int) image.Image
	// Stop ends the clip once it returns true, checked after each captured
	// frame so the finishing state is included. A run of unknown length — a
	// game that ends when it ends — uses this instead of a frame count. Nil
	// runs on to Frames or MaxSteps.
	Stop func(step int) bool
}

// Record runs the clip, adding frames to rec, and reports how many it
// captured. It stops at the frame count, the step cap, or when the recorder
// reports itself done — whichever comes first.
func (c Clip) Record(rec *record.Recorder) (int, error) {
	if err := c.validate(); err != nil {
		return 0, err
	}
	every := max(c.Every, 1)
	captured, rolling := 0, c.Ready == nil
	for step := 0; c.moreSteps(step) && !c.enough(captured) && !rec.Done(); step++ {
		if err := c.Step(step); err != nil {
			return captured, err
		}
		// Short-circuits when Ready is nil, since rolling then starts true.
		rolling = rolling || c.Ready(step)
		if !rolling || step%every != 0 {
			continue
		}
		rec.Add(c.Frame(step))
		captured++
		if c.Stop != nil && c.Stop(step) {
			break
		}
	}
	return captured, nil
}

// validate reports why the clip could not run, if it could not.
func (c Clip) validate() error {
	if c.Step == nil || c.Frame == nil {
		return errors.New("demo: Clip needs both Step and Frame")
	}
	if c.Frames <= 0 && c.MaxSteps <= 0 && c.Stop == nil {
		return errors.New("demo: Clip needs Frames, MaxSteps or Stop to terminate")
	}
	return nil
}

// moreSteps reports whether the step cap still allows another step.
func (c Clip) moreSteps(step int) bool { return c.MaxSteps <= 0 || step < c.MaxSteps }

// enough reports whether the frame count has been reached.
func (c Clip) enough(captured int) bool { return c.Frames > 0 && captured >= c.Frames }

// Montage tiles cells into a grid of the given column count, separated and
// bordered by gap pixels of bg. Cells are laid out left to right, top to
// bottom, each in a box the size of the largest cell, so a short final row
// stays aligned. It returns nil when there are no cells.
//
// Games use it for contact sheets — a row of weapon viewmodels, a grid of
// generated levels — that read better together than as separate files.
func Montage(cells []image.Image, cols, gap int, bg color.Color) *image.RGBA {
	if len(cells) == 0 {
		return nil
	}
	cols = max(cols, 1)
	gap = max(gap, 0)

	cellW, cellH := 0, 0
	for _, c := range cells {
		b := c.Bounds()
		cellW, cellH = max(cellW, b.Dx()), max(cellH, b.Dy())
	}
	rows := (len(cells) + cols - 1) / cols
	out := image.NewRGBA(image.Rect(0, 0,
		cols*cellW+(cols+1)*gap,
		rows*cellH+(rows+1)*gap,
	))
	draw.Draw(out, out.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
	for i, c := range cells {
		b := c.Bounds()
		x := gap + (i%cols)*(cellW+gap)
		y := gap + (i/cols)*(cellH+gap)
		draw.Draw(out, image.Rect(x, y, x+b.Dx(), y+b.Dy()), c, b.Min, draw.Src)
	}
	return out
}

// Downscale shrinks src by an integer factor, averaging each factor×factor
// block of source pixels into one. A factor below two returns src unchanged.
//
// This is the filter for a still that will be looked at: a montage cell, a
// screenshot, a documentation frame. It is deliberately not the one
// [record.Recorder] applies to captured frames, which point-samples instead —
// averaging mixes new colours that were never in the scene, and a frame-diffed
// GIF needs unchanged regions to quantise to byte-identical palette entries.
func Downscale(src image.Image, factor int) image.Image {
	if factor < 2 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx()/factor, b.Dy()/factor
	out := image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	n := uint32(factor * factor)
	for y := range h {
		for x := range w {
			var sr, sg, sb, sa uint32
			for dy := range factor {
				for dx := range factor {
					r, g, bl, a := src.At(b.Min.X+x*factor+dx, b.Min.Y+y*factor+dy).RGBA()
					// RGBA returns 16-bit premultiplied channels; drop to 8.
					sr, sg, sb, sa = sr+r>>8, sg+g>>8, sb+bl>>8, sa+a>>8
				}
			}
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(sr / n), G: uint8(sg / n), B: uint8(sb / n), A: uint8(sa / n),
			})
		}
	}
	return out
}

// Ramp builds a GIF palette by fanning each base colour into steps brightness
// levels, from dim to full. A palette made of the scene's own colours
// quantises cleanly without dithering, which keeps a frame-diffed GIF small;
// pass the result to [record.WithPalette].
//
// The palette opens with opaque black so unlit pixels have an exact match.
// Fewer than one step, or no bases, yields just that black.
func Ramp(bases []color.RGBA, steps int) color.Palette {
	pal := color.Palette{color.RGBA{A: 255}}
	if steps < 1 {
		return pal
	}
	for _, b := range bases {
		for s := range steps {
			// Start at a tenth rather than zero: the darkest shade should still
			// carry the base hue instead of collapsing to black.
			f := 0.1 + 0.9*float64(s)/float64(max(steps-1, 1))
			pal = append(pal, color.RGBA{
				R: uint8(float64(b.R) * f),
				G: uint8(float64(b.G) * f),
				B: uint8(float64(b.B) * f),
				A: 255,
			})
		}
	}
	return pal
}
