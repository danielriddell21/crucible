package narrate_test

import (
	"testing"
	"time"

	"github.com/danielriddell21/crucible/hud"
	"github.com/danielriddell21/crucible/narrate"
	"github.com/danielriddell21/crucible/status"
)

const personasJSON = `{
  "version": "0.1",
  "default": "tester",
  "personas": [
    {
      "id": "tester",
      "name": "Tester",
      "description": "A terse test persona.",
      "style": {"tone": "calm", "energy": "low", "verbosity": "short"},
      "rules": ["React to the event in one short line."],
      "constraints": {"max_words": 10, "max_sentences": 1},
      "examples": [
        {"event": "cleared", "output": "Level {level} cleared."}
      ]
    }
  ]
}`

type cue struct {
	kind  string
	level int
}

func scripted() status.Source[cue] {
	return status.Func[cue](func(c cue, emit func(status.Line)) {
		switch c.kind {
		case "cleared":
			emit(status.Line{Text: "scripted clear", Channel: hud.Notice, Frames: 100})
		case "debug":
			emit(status.Line{Text: "telemetry: raw", Channel: hud.Diagnostic, Frames: 100})
		}
	})
}

func newSource(t *testing.T) *narrate.Source[cue] {
	t.Helper()
	s, err := narrate.New(
		narrate.Config{Personas: []byte(personasJSON), Persona: "tester", MaxWords: 10, Timeout: 2 * time.Second},
		scripted(),
		func(c cue) string { return c.kind },
		func(c cue) map[string]any { return map[string]any{"level": c.level} },
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestNoticeEmitsOnNoticeChannel(t *testing.T) {
	s := newSource(t)
	var line status.Line
	var got bool
	s.Request(cue{kind: "cleared", level: 3}, func(l status.Line) { line, got = l, true })
	if !got {
		t.Fatal("no line emitted for a scripted notice")
	}
	if line.Channel != hud.Notice || line.Frames != 100 || line.Text == "" {
		t.Fatalf("line = %+v", line)
	}
}

func TestDiagnosticPassesThroughUntouched(t *testing.T) {
	s := newSource(t)
	var line status.Line
	s.Request(cue{kind: "debug"}, func(l status.Line) { line = l })
	if line.Text != "telemetry: raw" || line.Channel != hud.Diagnostic {
		t.Fatalf("line = %+v", line)
	}
}

func TestScriptedSilenceStaysSilent(t *testing.T) {
	s := newSource(t)
	called := false
	s.Request(cue{kind: "nothing"}, func(status.Line) { called = true })
	if called {
		t.Fatal("emit ran for a cue the scripted source ignored")
	}
}

func TestBadPersonasFails(t *testing.T) {
	_, err := narrate.New(
		narrate.Config{Personas: []byte("{not json"), Persona: "tester"},
		scripted(),
		func(c cue) string { return c.kind },
		func(cue) map[string]any { return nil },
	)
	if err == nil {
		t.Fatal("New must reject a corrupt personas document")
	}
}
