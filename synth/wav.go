package synth

import "encoding/binary"

// wavHeaderSize is the length of the canonical RIFF/WAVE header for
// uncompressed PCM.
const wavHeaderSize = 44

// WAV wraps rendered PCM in a RIFF/WAVE header, describing it with the
// package's own format constants.
//
// [Render] and the cue shapes above it produce raw interleaved samples, which
// suits an Ebiten or oto player because those take a stream of them directly.
// Most other front-ends do not: raylib, SDL and the browser all load an encoded
// file, and will not accept bare samples however well described. Forty-four
// bytes of header is the whole difference, so this saves every non-Ebiten
// caller from carrying its own copy of a fiddly little spec.
//
//	sound := rl.LoadSoundFromWave(rl.LoadWaveFromMemory(".wav", w, int32(len(w))))
//
// The returned slice is a fresh buffer; pcm is not retained or modified.
func WAV(pcm []byte) []byte {
	out := make([]byte, wavHeaderSize+len(pcm))
	put := binary.LittleEndian

	copy(out[0:], "RIFF")
	// Everything after this field, which is the header remainder plus the data.
	put.PutUint32(out[4:], uint32(wavHeaderSize-8+len(pcm)))
	copy(out[8:], "WAVE")

	copy(out[12:], "fmt ")
	put.PutUint32(out[16:], 16) // the PCM format chunk is sixteen bytes
	put.PutUint16(out[20:], 1)  // uncompressed
	put.PutUint16(out[22:], ChannelCount)
	put.PutUint32(out[24:], SampleRate)
	put.PutUint32(out[28:], SampleRate*BytesPerFrame) // byte rate
	put.PutUint16(out[32:], BytesPerFrame)            // block align
	put.PutUint16(out[34:], BitDepthInBytes*8)

	copy(out[36:], "data")
	put.PutUint32(out[40:], uint32(len(pcm)))
	copy(out[wavHeaderSize:], pcm)
	return out
}
