package synth_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/synth"
)

func frames(pcm []byte) int { return len(pcm) / synth.BytesPerFrame }

func sample(pcm []byte, i int) (l, r int16) {
	off := i * synth.BytesPerFrame
	l = int16(uint16(pcm[off]) | uint16(pcm[off+1])<<8)
	r = int16(uint16(pcm[off+2]) | uint16(pcm[off+3])<<8)
	return l, r
}

func TestRenderLengthAndLayout(t *testing.T) {
	pcm := synth.Render(0.5, func(float64) float64 { return 0.25 })
	if got, want := frames(pcm), synth.SampleRate/2; got != want {
		t.Fatalf("frames = %d, want %d", got, want)
	}
	l, r := sample(pcm, 100)
	if l != r {
		t.Fatalf("mono render must duplicate channels: %d vs %d", l, r)
	}
	if l < 8000 || l > 8300 { // 0.25 * 32767 ≈ 8191
		t.Fatalf("sample = %d", l)
	}
}

func TestRenderClips(t *testing.T) {
	pcm := synth.Render(0.01, func(float64) float64 { return 2 })
	if l, _ := sample(pcm, 10); l != 32767 {
		t.Fatalf("over-range sample = %d, want clipped 32767", l)
	}
	pcm = synth.Render(0.01, func(float64) float64 { return -2 })
	if l, _ := sample(pcm, 10); l != -32767 {
		t.Fatalf("under-range sample = %d, want clipped -32767", l)
	}
}

func TestEnvelopes(t *testing.T) {
	if synth.Env(0, 10) != 1 {
		t.Error("Env(0) must be 1")
	}
	if synth.Env(1, 10) >= synth.Env(0.5, 10) {
		t.Error("Env must decay")
	}
	if synth.Attack(0, 8) != 0 || synth.Attack(1, 8) != 1 {
		t.Error("Attack endpoints wrong")
	}
	if got := synth.Attack(0.0625, 8); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("Attack midpoint = %v", got)
	}
}

func TestNoiseDeterministic(t *testing.T) {
	a, b := synth.Noise(1, 2), synth.Noise(1, 2)
	for range 10 {
		va, vb := a(), b()
		if va != vb {
			t.Fatal("same seed must give same noise")
		}
		if va < -1 || va > 1 {
			t.Fatalf("noise out of range: %v", va)
		}
	}
}

func TestPanConstantPower(t *testing.T) {
	for _, bearing := range []float64{0, math.Pi / 2, -math.Pi / 2, math.Pi} {
		l, r := synth.Pan(bearing)
		if p := l*l + r*r; math.Abs(p-1) > 1e-9 {
			t.Errorf("Pan(%v): power = %v", bearing, p)
		}
	}
	// Straight ahead and directly behind are centred.
	for _, bearing := range []float64{0, math.Pi} {
		l, r := synth.Pan(bearing)
		if math.Abs(l-r) > 1e-9 {
			t.Errorf("Pan(%v) not centred: %v vs %v", bearing, l, r)
		}
	}
	// Hard right favours the right ear.
	l, r := synth.Pan(math.Pi / 2)
	if r <= l {
		t.Errorf("Pan(right): l=%v r=%v", l, r)
	}
}

func TestPannedScalesChannels(t *testing.T) {
	src := synth.Render(0.01, func(float64) float64 { return 0.5 })
	out := synth.Panned(src, 0, 1)
	l, r := sample(out, 10)
	if l != 0 {
		t.Fatalf("left must be muted, got %d", l)
	}
	sl, _ := sample(src, 10)
	if r != sl {
		t.Fatalf("right must be untouched: %d vs %d", r, sl)
	}
	if len(out) != len(src) {
		t.Fatal("Panned must preserve length")
	}
}

func TestCueShapes(t *testing.T) {
	cases := []struct {
		name string
		pcm  []byte
		dur  float64
	}{
		{"Blip", synth.Blip(880, 0.05, 60), 0.05},
		{"TwoTone", synth.TwoTone(660, 990, 0.08, 0.18, 22), 0.18},
		{"Arpeggio", synth.Arpeggio([]float64{523, 659, 784}, 0.15, 0.45, 10), 0.45},
		{"Slide", synth.Slide(300, -360, 60, 0.6, 4), 0.6},
		{"Rumble", synth.Rumble(90, 40, 0.4, 5, 6), 0.4},
		{"Thud", synth.Thud(150, 0.1, 55, 3, 4), 0.1},
		{"Drone", synth.Drone([]synth.Voice{{Freq: 55, Amp: 0.18}, {Freq: 110, Amp: 0.07}}, 0.5, 0.4, 1), 1},
	}
	for _, c := range cases {
		if got, want := frames(c.pcm), int(c.dur*synth.SampleRate); got != want {
			t.Errorf("%s: frames = %d, want %d", c.name, got, want)
			continue
		}
		sum := 0
		for _, b := range c.pcm {
			sum += int(b)
		}
		if sum == 0 {
			t.Errorf("%s rendered silence", c.name)
		}
	}
}

func TestArpeggioEmptyNotes(t *testing.T) {
	pcm := synth.Arpeggio(nil, 0.1, 0.2, 10)
	if got, want := frames(pcm), int(0.2*synth.SampleRate); got != want {
		t.Fatalf("frames = %d, want %d", got, want)
	}
	for _, b := range pcm {
		if b != 0 {
			t.Fatal("empty arpeggio must be silent")
		}
	}
}
