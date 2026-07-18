package status_test

import (
	"testing"

	"github.com/danielriddell21/crucible/hud"
	"github.com/danielriddell21/crucible/status"
)

type cue struct{ kind string }

func TestFuncSource(t *testing.T) {
	src := status.Func[cue](func(c cue, emit func(status.Line)) {
		if c.kind == "loud" {
			emit(status.Line{Text: "heard", Channel: hud.Notice, Frames: 10})
		}
	})

	var got []status.Line
	src.Request(cue{kind: "loud"}, func(l status.Line) { got = append(got, l) })
	src.Request(cue{kind: "quiet"}, func(l status.Line) { got = append(got, l) })
	if len(got) != 1 || got[0].Text != "heard" {
		t.Fatalf("lines = %+v", got)
	}
}

func TestEmitPostsToOverlay(t *testing.T) {
	o := hud.New()
	emit := status.Emit(o)
	emit(status.Line{Text: "hello", Channel: hud.Notice, Frames: 5})
	text, ch, ok := o.Active()
	if !ok || text != "hello" || ch != hud.Notice {
		t.Fatalf("overlay = %q %v %v", text, ch, ok)
	}
}
