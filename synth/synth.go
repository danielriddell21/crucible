// Package synth renders the family's procedural sound effects and loops as
// interleaved 16-bit stereo PCM at [SampleRate], ready for an Ebiten/oto
// audio player.
//
// [Render] is the core loop: it samples a generator function over a
// duration, clips, and interleaves. [Sine], [Noise], [Env], and [Attack]
// are the building blocks; [Pan] and [Panned] place a finished sound in
// the stereo field. On top of those sit the parameterised cue shapes the
// games share — [Blip], [TwoTone], [Arpeggio], [Slide], [Rumble], [Thud],
// and [Drone] — while each game's cue enums and sound design stay its own.
//
// Everything is deterministic: noise takes explicit seeds, so a cue
// renders sample-identical everywhere.
package synth

import (
	"math"
	"math/rand/v2"
)

// Audio format constants shared by every player in the family.
const (
	// SampleRate is the render and playback rate in Hz.
	SampleRate = 44100
	// ChannelCount is the number of interleaved channels (stereo).
	ChannelCount = 2
	// BitDepthInBytes is the sample width (16-bit).
	BitDepthInBytes = 2
	// BytesPerFrame is the size of one interleaved sample frame.
	BytesPerFrame = ChannelCount * BitDepthInBytes
)

// Render synthesises dur seconds of audio by sampling gen at each frame
// time. gen returns an amplitude in [-1, 1]; values outside are clipped.
// Both stereo channels carry the same signal — use [Panned] to place a
// sound.
func Render(dur float64, gen func(t float64) float64) []byte {
	n := int(dur * SampleRate)
	buf := make([]byte, n*BytesPerFrame)
	for i := range n {
		v := gen(float64(i) / SampleRate)
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		s := int16(v * 32767)
		lo, hi := byte(s), byte(s>>8)
		off := i * BytesPerFrame
		buf[off], buf[off+1] = lo, hi   // left
		buf[off+2], buf[off+3] = lo, hi // right
	}
	return buf
}

// Env is an exponential decay envelope: 1 at t=0 falling at the given rate.
func Env(t, decay float64) float64 { return math.Exp(-t * decay) }

// Attack is a linear rise envelope: 0 at t=0 reaching 1 at t=1/rate.
func Attack(t, rate float64) float64 { return math.Min(1, t*rate) }

// Sine is a sine oscillator at freq Hz evaluated at time t.
func Sine(freq, t float64) float64 { return math.Sin(2 * math.Pi * freq * t) }

// Noise returns a deterministic white-noise sampler seeded with the given
// values. Each call to the sampler yields a value in [-1, 1].
func Noise(seed1, seed2 uint64) func() float64 {
	r := rand.New(rand.NewPCG(seed1, seed2))
	return func() float64 { return r.Float64()*2 - 1 }
}

// Pan converts a bearing relative to the listener's facing (0 = dead ahead,
// positive to the right) into constant-power left/right channel gains in
// [0, 1]. Sounds ahead or behind sit centred; sounds to a side swing toward
// that ear.
func Pan(bearing float64) (left, right float64) {
	// Fold front/back onto the same left-right axis: a sound directly behind
	// pans the same as one directly ahead (centred).
	x := math.Sin(bearing) // -1 hard left, +1 hard right
	angle := (x + 1) * (math.Pi / 4)
	return math.Cos(angle), math.Sin(angle)
}

// Panned returns a copy of interleaved 16-bit stereo PCM with the left and
// right channels scaled by the given gains.
func Panned(pcm []byte, left, right float64) []byte {
	out := make([]byte, len(pcm))
	for i := 0; i+BytesPerFrame <= len(pcm); i += BytesPerFrame {
		l := int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8)
		r := int16(uint16(pcm[i+2]) | uint16(pcm[i+3])<<8)
		l = int16(float64(l) * left)
		r = int16(float64(r) * right)
		out[i], out[i+1] = byte(l), byte(uint16(l)>>8)
		out[i+2], out[i+3] = byte(r), byte(uint16(r)>>8)
	}
	return out
}
