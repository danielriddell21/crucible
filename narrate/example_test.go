package narrate_test

import (
	"log"

	"github.com/danielriddell21/crucible/narrate"
	"github.com/danielriddell21/crucible/status"
)

// ExampleNew decorates a game's scripted source with narrata rewording:
// Notice lines may be rephrased by the persona, and the scripted wording
// stands in whenever generation declines. The cue type belongs to the
// game, and personasJSON would be a go:embed of its personas.json.
func ExampleNew() {
	scripted := status.Func[cue](func(c cue, emit func(status.Line)) {
		// ... the game's scripted table decides what to say ...
	})

	src, err := narrate.New(
		narrate.Config{Personas: []byte(personasJSON), Persona: "tester", MaxWords: 14},
		scripted,
		func(c cue) string { return c.kind },
		func(c cue) map[string]any { return map[string]any{"level": c.level + 1} },
	)
	if err != nil {
		log.Fatal(err)
	}
	defer src.Close()

	src.Request(cue{kind: "level_complete", level: 6}, func(l status.Line) {
		// post l to the HUD overlay
	})
}
