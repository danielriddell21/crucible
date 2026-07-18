package status_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/hud"
	"github.com/danielriddell21/crucible/status"
)

// Example wires a scripted source to an overlay. The cue type belongs to
// the game; here it is a tiny struct with a kind and a depth.
func Example() {
	type cue struct {
		kind  string
		level int
	}

	scripted := status.Func[cue](func(c cue, emit func(status.Line)) {
		if c.kind == "exit" {
			emit(status.Line{
				Text:    fmt.Sprintf("Level %d cleared.", c.level+1),
				Channel: hud.Notice,
				Frames:  150,
			})
		}
	})

	overlay := hud.New()
	scripted.Request(cue{kind: "exit", level: 6}, status.Emit(overlay))
	scripted.Request(cue{kind: "step", level: 6}, status.Emit(overlay)) // stays silent

	text, _, _ := overlay.Active()
	fmt.Println(text)
	// Output: Level 7 cleared.
}
