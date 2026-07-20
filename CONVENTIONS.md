# Conventions

Structure shared across the tool family — unum, fiat-lux, galapagos, pandemonium,
vivarium, toolshed, narrata, gambit, rubix, hegemony, nemesis — so they read like
they were written by one person. Each repo records only the conventions it follows;
crucible is the **engine library** the family's GUI conventions build on. unum is
the CLI reference; rubix and vivarium are the GUI references.

These cover the *shape* of the engine and how apps consume it, not the behaviour
inside them.

## Engine library

crucible is a flat library: one package per engine concern at the module root
(`geom`, `hud`, `canvas`, `menu`, `store`, `status`, `narrate`, `telemetry`,
`synth`, `record`, `hub`, `worldgen`, `raycast`, `camera`), no `cmd/`
and no `internal/`.

- **App types stay in the apps.** Packages that relay or store app-defined
  values (`hub`, `telemetry`, `status`, `narrate`) are generic over them; the
  message, event, and cue vocabularies belong to each game.
- **Ebiten stays at the edge.** Only `menu` (input polling) and `camera`
  (GeoM) import Ebiten. Everything else — rendering math, PCM synthesis,
  world generation, persistence — is plain Go, testable without a display.
- **Full godoc.** Every exported symbol carries a doc comment; the revive
  `exported` rule enforces it.
- **Deterministic by seed.** Anything random (`worldgen`, `synth` noise)
  takes explicit seeds so a seed reproduces the same result everywhere.

## How apps consume crucible

The family's app-side conventions are unchanged; crucible supplies their
machinery:

- **CLI entrypoint.** The command tree stays in each repo's `internal/cli`
  with a thin `cmd/<name>/main.go`; the root command and completion stay
  there too — crucible is engine code. Do not add a bespoke `version`
  subcommand.
- **Ebiten GUI.** `internal/gui` remains the only app package that imports
  Ebiten, exposing `func Run(cfg Config) error` and `func Available() bool`
  in both builds, with the `//go:build ebiten` stub policy for repos that
  have a real headless mode. The window, `ebiten.Game`, and input decomposition
  live there; crucible's `menu`, `canvas`, `hud`, `record`, and `hub` slot in
  behind that seam.
- **Multi-window.** Repos with a leader/child window mode route their `Msg`
  type through `hub` with a `Route` policy instead of hand-rolled coordination.
