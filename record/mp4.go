package record

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNoFFmpeg reports that MP4 encoding was requested but the ffmpeg binary is
// not on PATH. GIF and PNG output need no external tools; only video does.
var ErrNoFFmpeg = errors.New("record: ffmpeg not found on PATH (needed for MP4 output)")

// IsVideoPath reports whether path names a video file, i.e. one this package
// encodes with ffmpeg rather than as a GIF. Front-ends use it to decide the
// output format from a --record path alone.
func IsVideoPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".mp4")
}

// EncodeMP4 pipes raw w×h RGBA frames to ffmpeg and writes an H.264 MP4 at
// path, played back at fps frames per second. Video stays small where a long
// GIF would not, which is why the family's longer demo clips use it. It
// returns [ErrNoFFmpeg] when ffmpeg is unavailable, so a caller can fall back
// to a GIF or report the missing dependency plainly.
func EncodeMP4(path string, frames [][]byte, w, h int, fps float64) error {
	if len(frames) == 0 {
		return fmt.Errorf("record: no frames to encode to %q", path)
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return ErrNoFFmpeg
	}
	if fps <= 0 {
		fps = 1
	}
	cmd := exec.CommandContext(context.Background(), "ffmpeg",
		"-y", "-loglevel", "error",
		"-f", "rawvideo", "-pixel_format", "rgba",
		"-video_size", fmt.Sprintf("%dx%d", w, h),
		"-framerate", fmt.Sprintf("%.4f", fps),
		"-i", "-",
		// yuv420p keeps the file playable in browsers and editors; faststart
		// puts the index up front so it streams without a full download.
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart",
		path,
	)
	var buf bytes.Buffer
	for _, fr := range frames {
		buf.Write(fr)
	}
	cmd.Stdin = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("record: run ffmpeg for %q: %w", path, err)
	}
	return nil
}
