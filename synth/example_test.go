package synth_test

import (
	"fmt"
	"math"

	"github.com/danielriddell21/crucible/synth"
)

// ExampleRender synthesises half a second of a decaying 440 Hz tone — the
// bones of every cue in the family.
func ExampleRender() {
	pcm := synth.Render(0.5, func(t float64) float64 {
		return 0.5 * synth.Sine(440, t) * synth.Env(t, 6)
	})
	fmt.Println(len(pcm) / synth.BytesPerFrame)
	// Output: 22050
}

// ExamplePan places a sound by its bearing from the listener's facing:
// ahead is centred, hard right favours the right ear.
func ExamplePan() {
	l, r := synth.Pan(0)
	fmt.Printf("ahead  %.2f %.2f\n", l, r)
	l, r = synth.Pan(math.Pi / 2)
	fmt.Printf("right  %.2f %.2f\n", l, r)
	// Output:
	// ahead  0.71 0.71
	// right  0.00 1.00
}

// ExampleArpeggio renders the family's conventional secret-found fanfare:
// a C5-E5-G5 arpeggio.
func ExampleArpeggio() {
	pcm := synth.Arpeggio([]float64{523, 659, 784}, 0.15, 0.45, 10)
	fmt.Printf("%.2fs\n", float64(len(pcm)/synth.BytesPerFrame)/synth.SampleRate)
	// Output: 0.45s
}
