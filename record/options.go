package record

import "github.com/spf13/pflag"

// Options bundles the recording settings the family's front-ends expose as
// flags: where to write the GIF, the playback rate, the downscale factor, and
// how many frames to capture before exiting. It is the shared shape behind
// every app's --record flags; [Options.AddFlags] registers them and [New]
// turns the result into a [Recorder].
type Options struct {
	// Path is the output GIF path. An empty path means "do not record".
	Path string
	// FPS is the recording's playback rate in frames per second.
	FPS int
	// Scale downscales each captured frame by this integer factor.
	Scale int
	// Frames caps the recording; zero means unlimited.
	Frames int
}

// Canonical flag defaults, used when a field is left at its zero value.
const (
	defaultFPS    = 30
	defaultScale  = 1
	defaultFrames = 600
)

// AddFlags registers the standard --record, --record-fps, --record-scale and
// --record-frames flags on fs, bound to o. A front-end wires them with
// cmd.Flags() from its own cobra command, so crucible supplies the flags
// without owning the command tree. Pre-set a field before calling to change
// that flag's default (e.g. Options{Scale: 2}); a zero field uses the
// canonical default.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fps, scale, frames := nonZero(o.FPS, defaultFPS), nonZero(o.Scale, defaultScale), nonZero(o.Frames, defaultFrames)
	fs.StringVar(&o.Path, "record", o.Path, "record the run to this GIF path, then exit")
	fs.IntVar(&o.FPS, "record-fps", fps, "recording playback rate in frames per second")
	fs.IntVar(&o.Scale, "record-scale", scale, "downscale factor for the recording")
	fs.IntVar(&o.Frames, "record-frames", frames, "frames to capture before exiting")
}

// AddPacedFlags registers the --record and --record-frames flags on fs, bound
// to o, for a recorder whose playback is paced by a fixed per-frame delay
// ([WithFrameDelay]) rather than a real-time frame rate. It omits --record-fps
// and --record-scale, which such a recorder ignores: the frame delay overrides
// the fps-derived timing, and these event-paced demos capture at full
// resolution. A zero Frames leaves the cap to the caller (its own default or
// the whole run); pre-set it to change the flag's default.
func (o *Options) AddPacedFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Path, "record", o.Path, "record the run to this GIF path, then exit")
	fs.IntVar(&o.Frames, "record-frames", o.Frames, "frames to capture before exiting")
}

// Recording reports whether a recording was requested (a path was set).
func (o Options) Recording() bool { return o.Path != "" }

// New returns a recorder configured from o. Extra options ([WithPalette],
// [WithFrameDiff], …) still apply.
func New(o Options, opts ...Option) *Recorder {
	return NewRecorder(o.FPS, o.Scale, o.Frames, opts...)
}

func nonZero(v, def int) int {
	if v != 0 {
		return v
	}
	return def
}
