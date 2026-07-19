package synth

import "math"

// Blip is a short pure tone with a fast decay — the family's menu tick.
func Blip(freq, dur, decay float64) []byte {
	return Render(dur, func(t float64) float64 {
		return 0.4 * Sine(freq, t) * Env(t, decay)
	})
}

// TwoTone plays f1 then jumps to f2 at the split point — the conventional
// pickup chirp.
func TwoTone(f1, f2, split, dur, decay float64) []byte {
	return Render(dur, func(t float64) float64 {
		f, local := f1, t
		if t > split {
			f, local = f2, t-split
		}
		return 0.5 * Sine(f, t) * Env(local, decay)
	})
}

// Arpeggio steps through notes at the given interval, restarting the decay
// envelope on each note — the conventional secret/fanfare shape.
func Arpeggio(notes []float64, step, dur, decay float64) []byte {
	if len(notes) == 0 {
		return Render(dur, func(float64) float64 { return 0 })
	}
	return Render(dur, func(t float64) float64 {
		idx := int(t / step)
		if idx >= len(notes) {
			idx = len(notes) - 1
		}
		local := t - float64(idx)*step
		return 0.45 * Sine(notes[idx], t) * Env(local, decay)
	})
}

// Slide sweeps a tone from f0 by rate Hz per second, clamped at floor — the
// conventional downward death groan when rate is negative.
func Slide(f0, rate, floor, dur, decay float64) []byte {
	return Render(dur, func(t float64) float64 {
		f := f0 + rate*t
		if f < floor {
			f = floor
		}
		return 0.5 * Sine(f, t) * Env(t, decay)
	})
}

// Rumble is a rising low tone with a noise grit layer and a brief attack —
// the conventional door sound.
func Rumble(base, rise, dur float64, seed1, seed2 uint64) []byte {
	noise := Noise(seed1, seed2)
	return Render(dur, func(t float64) float64 {
		e := Attack(t, 8) * Env(t, 5) // brief rise, slow fall
		rumble := Sine(base+rise*t, t)
		grit := noise() * 0.2
		return (0.55*rumble + grit) * e
	})
}

// Thud is a low tone mixed with noise under a hard decay — the conventional
// impact sound.
func Thud(freq, dur, decay float64, seed1, seed2 uint64) []byte {
	noise := Noise(seed1, seed2)
	return Render(dur, func(t float64) float64 {
		e := Env(t, decay)
		return (0.6*Sine(freq, t) + 0.4*noise()) * e
	})
}

// Drone layers the given voices (frequency, amplitude pairs) under a slow
// sine LFO — the conventional ambient bed. The LFO rate is in Hz.
func Drone(voices []Voice, lfoRate, lfoDepth, dur float64) []byte {
	return Render(dur, func(t float64) float64 {
		lfo := (1 - lfoDepth) + lfoDepth*math.Sin(2*math.Pi*lfoRate*t)
		v := 0.0
		for _, voice := range voices {
			v += voice.Amp * Sine(voice.Freq, t)
		}
		return v * lfo
	})
}

// Voice is one layer of a Drone: a frequency and its amplitude.
type Voice struct {
	Freq, Amp float64
}
