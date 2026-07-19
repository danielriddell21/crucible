// Package narrate wraps a scripted [status.Source] with a narrata
// narration engine: player-facing [hud.Notice] lines are rewritten by a
// persona, while the scripted wording stands in whenever generation
// declines, fails, or times out. Diagnostic lines pass through untouched.
//
// [New] starts the engine behind a [Source] from a [Config] carrying the
// game's embedded personas document; the game supplies two functions
// mapping its cue type to a persona event name and its data payload.
// [Source.Close] shuts the engine down.
//
// The [github.com/danielriddell21/narrata] dependency is confined to this
// package, so games that only want scripted lines never link it.
package narrate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/danielriddell21/narrata"

	"github.com/danielriddell21/crucible/hud"
	"github.com/danielriddell21/crucible/status"
)

// Config configures the narration engine behind a Source.
type Config struct {
	// Personas is the raw personas.json document, typically go:embed-ded by
	// the game.
	Personas []byte
	// Persona is the persona ID used for every request.
	Persona string
	// MaxWords caps the generated line length. Zero means no cap.
	MaxWords int
	// Timeout bounds each generation; the scripted line is used when it
	// expires. Zero defaults to two seconds.
	Timeout time.Duration
	// MaxConcurrent bounds in-flight generations. Zero defaults to two.
	MaxConcurrent int
}

// Source decorates a scripted status.Source with narrata rewording. C is
// the game's cue type.
type Source[C any] struct {
	engine  *narrata.Engine
	base    status.Source[C]
	event   func(C) string
	data    func(C) map[string]any
	persona string
	words   int
	timeout time.Duration
	tmp     string
}

// New starts a narration engine and returns a [Source] that rewrites base's
// [hud.Notice] lines. The event function names the cue for the persona
// (e.g. "level_complete"); data supplies the placeholders the persona may
// use.
func New[C any](cfg Config, base status.Source[C], event func(C) string, data func(C) map[string]any) (*Source[C], error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 2
	}
	tmp, err := writeTemp(cfg.Personas)
	if err != nil {
		return nil, err
	}
	engine, err := narrata.New(narrata.Config{
		PersonasPath:   tmp,
		DefaultPersona: cfg.Persona,
		Text:           narrata.TextConfig{Backend: "native"},
		MaxConcurrent:  cfg.MaxConcurrent,
		Timeout:        cfg.Timeout,
	})
	if err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("start narration engine: %w", err)
	}
	return &Source[C]{
		engine:  engine,
		base:    base,
		event:   event,
		data:    data,
		persona: cfg.Persona,
		words:   cfg.MaxWords,
		timeout: cfg.Timeout,
		tmp:     tmp,
	}, nil
}

// Request implements status.Source. The scripted source speaks first; when
// it produces a Notice line, the engine may reword it.
func (s *Source[C]) Request(cue C, emit func(status.Line)) {
	var line status.Line
	var got bool
	s.base.Request(cue, func(l status.Line) { line, got = l, true })
	if !got {
		return // the scripted policy stayed silent for this cue
	}
	if line.Channel != hud.Notice {
		emit(line) // diagnostics are never rewritten
		return
	}
	if text, ok := s.generate(context.Background(), cue); ok {
		line.Text = text
	}
	emit(line) // the scripted wording stands in when nothing was generated
}

func (s *Source[C]) generate(ctx context.Context, cue C) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	res, err := s.engine.Generate(ctx, narrata.Request{
		PersonaID:   s.persona,
		Event:       s.event(cue),
		Data:        s.data(cue),
		Output:      narrata.OutputText,
		Constraints: narrata.Constraints{MaxWords: s.words},
	})
	if err != nil || res.Silent || res.Text == "" {
		return "", false
	}
	return res.Text, true
}

// Close shuts down the narration engine and removes the temporary personas
// file.
func (s *Source[C]) Close() error {
	if s.tmp != "" {
		_ = os.Remove(s.tmp)
	}
	if err := s.engine.Close(); err != nil {
		return fmt.Errorf("close narration engine: %w", err)
	}
	return nil
}

func writeTemp(b []byte) (string, error) {
	f, err := os.CreateTemp("", "crucible-personas-*.json")
	if err != nil {
		return "", fmt.Errorf("create personas file: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("write personas file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("close personas file: %w", err)
	}
	return f.Name(), nil
}
