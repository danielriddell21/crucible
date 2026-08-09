package synth_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/danielriddell21/crucible/synth"
)

func TestWAVDescribesThePackageFormat(t *testing.T) {
	pcm := synth.Blip(440, 0.1, 6)
	w := synth.WAV(pcm)

	if got, want := len(w), len(pcm)+44; got != want {
		t.Fatalf("length = %d, want %d", got, want)
	}
	for _, tag := range []struct {
		at   int
		want string
	}{{0, "RIFF"}, {8, "WAVE"}, {12, "fmt "}, {36, "data"}} {
		if got := string(w[tag.at : tag.at+4]); got != tag.want {
			t.Errorf("tag at %d = %q, want %q", tag.at, got, tag.want)
		}
	}

	le := binary.LittleEndian
	if got := le.Uint16(w[22:]); got != synth.ChannelCount {
		t.Errorf("channels = %d, want %d", got, synth.ChannelCount)
	}
	if got := le.Uint32(w[24:]); got != synth.SampleRate {
		t.Errorf("sample rate = %d, want %d", got, synth.SampleRate)
	}
	if got := le.Uint32(w[28:]); got != synth.SampleRate*synth.BytesPerFrame {
		t.Errorf("byte rate = %d, want %d", got, synth.SampleRate*synth.BytesPerFrame)
	}
	if got := le.Uint16(w[32:]); got != synth.BytesPerFrame {
		t.Errorf("block align = %d, want %d", got, synth.BytesPerFrame)
	}
	if got := le.Uint16(w[34:]); got != synth.BitDepthInBytes*8 {
		t.Errorf("bit depth = %d, want %d", got, synth.BitDepthInBytes*8)
	}
}

func TestWAVSizesAgreeWithTheData(t *testing.T) {
	pcm := synth.Blip(440, 0.05, 6)
	w := synth.WAV(pcm)
	le := binary.LittleEndian

	if got, want := le.Uint32(w[40:]), uint32(len(pcm)); got != want {
		t.Errorf("data size = %d, want %d", got, want)
	}
	// The RIFF size counts everything after itself.
	if got, want := le.Uint32(w[4:]), uint32(len(w)-8); got != want {
		t.Errorf("riff size = %d, want %d", got, want)
	}
}

func TestWAVCarriesTheSamplesUntouched(t *testing.T) {
	pcm := synth.TwoTone(700, 500, 0.2, 0.4, 0)
	w := synth.WAV(pcm)
	if !bytes.Equal(w[44:], pcm) {
		t.Error("the samples were altered on the way into the file")
	}
}

func TestWAVDoesNotAliasItsInput(t *testing.T) {
	pcm := synth.Blip(440, 0.02, 6)
	w := synth.WAV(pcm)
	w[44] ^= 0xff
	if w[44] == pcm[0] {
		t.Error("the header wraps the caller's buffer instead of a copy")
	}
}

func TestWAVOfNothingIsStillAValidFile(t *testing.T) {
	w := synth.WAV(nil)
	if len(w) != 44 {
		t.Fatalf("length = %d, want a bare 44-byte header", len(w))
	}
	if binary.LittleEndian.Uint32(w[40:]) != 0 {
		t.Error("an empty file claims to hold data")
	}
}
