# Provenance

Where each crucible package came from: the files that were reviewed and
consolidated, and what deliberately stayed behind in each repo. Pandemonium
was reviewed at its `feat/audio-cues` branch; every other repo at `trunk`.

The last four rows (`view`, `rng`, `ring`, `paint`) came from a second pass
over the family. The rule of thumb, in order: if two or more repos carry the
same display-free logic, extract it; if it is nearly the same but carries
app-defined values, make the engine side generic and let each app keep its
vocabulary; otherwise leave it, and say why (see *Deliberately not
consolidated* below).

| Package | Consolidated from | Stays app-side |
|---|---|---|
| `geom` | vivarium `internal/geom/vec2.go`; the `Vec2`/`Coord`/`Room` types and clamp helpers repeated in pandemonium, nemesis, and hegemony | — |
| `hud` | pandemonium `internal/hud/overlay.go` and nemesis `internal/hud/overlay.go` (byte-identical) | — |
| `canvas` | nemesis `internal/gui/canvas.go` (pandemonium composes menus in its renderer the same way) | game-specific screens drawn on it |
| `menu` | nemesis `internal/gui/menu.go` + `menuflow.go`; pandemonium `internal/gui/menu.go` | the actual title/pause/settings menus, wired as `menu.Item` values |
| `store` | nemesis `internal/gui/settings.go` + `records.go`; pandemonium `internal/gui/settings.go` + `records.go` (load/save/path plumbing) | the `Settings`/`Records` structs, their clamping and summaries |
| `status` | pandemonium `internal/status/source.go` (`Line`, `Source` contract) | the cue vocabulary, band policy, and scripted tables |
| `narrate` | pandemonium `internal/phrasing/phrasing.go` | `personas.json`, the cue→event/data mapping |
| `telemetry` | pandemonium `internal/telemetry/bus.go`; nemesis `internal/telemetry/telemetry.go` (bus mechanics: fan-out, nil filtering, bounded feed) | event types, path/profile aggregation, feed wording |
| `synth` | pandemonium `internal/audio/synth.go`; nemesis `internal/audio/synth.go` (render loop, envelopes, pan, and the shared cue shapes) | cue enums, `CueFor` mappings, each game's sound design |
| `record` | rubix `internal/gui/record.go`; galapagos `internal/gui/record.go` (superset of both); gambit `internal/gui/record.go`; pandemonium `internal/gui/screenshot.go` | recording keybinds and drivers |
| `hub` | rubix `internal/cli/coord.go` + `internal/gui/link.go`; hegemony `internal/cli/coord.go` + `internal/gui/link.go`; nemesis `internal/cli/coord.go` + `internal/gui/link.go` | each app's `Msg` type and its `Route` policy |
| `worldgen` | pandemonium `internal/world/bsp.go` + `connectivity.go`; nemesis `internal/world/bsp.go` + `connectivity.go` | app-specific carving beyond the shared passes |
| `level` | the engine-owned world model: pandemonium `internal/world/level.go`, `tile.go`, `heights.go` (levels, dais, ceilings), `lowwall.go`, the lift ledge + kinematics, `generate.go` (attempt/validate pipeline, spawn/farthest-exit), `theme.go`, `sky.go`; nemesis `tile.go` (vent/console/locker → `TileVent`/`TileSwitch`/`TileCover`), `doors.go`, `vents.go` (**improved**: centred mouths instead of corner-biased, branching tree networks via nearest-carved-tunnel Dijkstra, depth bias keeping tunnels off wall faces) | items, markers, hazards, gates/keys, arenas, light moods and flicker, runtime door/lift state |
| `raycast` | nemesis `internal/render/walls.go` (DDA) + `sprites.go` (projection); pandemonium `internal/render/camera.go` + `columns.go` (the height-aware column walk, as `WalkColumn` over a painter interface) | texturing, shading, framebuffers, sliding-door column logic (built on the exported boundary helpers) |
| `camera` | vivarium `internal/gui/camera.go` | input wiring |
| `view` | galapagos `internal/core/camera.go` — the display-free pan/zoom math, adopted back as a type alias; `camera` now layers the Ebiten draw transform over it | the Ebiten draw transform (stays in `camera`) |
| `rng` | galapagos `internal/sim/rng.go` + hegemony `internal/sim/rng.go` (the PCG stream constructor), and gambit's three agent seeders | each app's stream ids and seed policy |
| `ring` | galapagos `internal/sim/telemetry.go` (`Ring[T]`) + vivarium's two hand-rolled bounded histories | what each ring holds |
| `paint` | pandemonium `internal/render/{walls,columns,sprites,tint}.go` + nemesis `internal/render/{renderer,hud}.go` (colour brightness scale + full-frame blend) | each game's distance-shading curve (art direction) |
| `keymap` | galapagos `Keymap []string` + `drawKeymap`; vivarium's two hard-coded overlay hint lines; rubix `segments` + `wrapHelp`; gambit's `hints` bar — the shared "key: action" control-bar layout (format, wrap, bottom-anchored stacking), kept display-free via a caller-supplied width measure | which keys each app binds and what they do |

## Deliberately not consolidated

Things that look shared but stay in each app, and why:

- **HUD / data panel rendering** (galapagos, vivarium, hegemony, gambit, rubix).
  Blocked two ways: only `menu` and `camera` may import Ebiten, so a panel
  renderer cannot become a package without breaking that rule; and the panels
  are genuinely bespoke (different data, different layouts) with no shared
  logic, only a shared look. The display-free kernels behind them did move: the
  rolling history a sparkline plots, as `ring`; and the control-hint bar's
  layout, as `keymap` (it takes a width-measure callback, so it positions the
  "key: action" hints without importing Ebiten). Text screens that do not sit
  on a live GL frame already have `canvas`.
- **Sprite frame / direction picking** (pandemonium, nemesis). Both choose a
  sprite per entity, but by different schemes (a `bands × dirs` face table vs.
  a state-and-time `alienFrame`). Gameplay presentation, app-side by the
  engine rules.
- **Notice / narration text** (pandemonium, nemesis). `notice()` and
  `Event.Line()` turn events into player-facing strings — the app-defined
  message vocabulary the engine must not grow. The mechanism is already shared
  (`telemetry.Bus`, `hud`, `status`, `narrate`); only the words stay home.
- **Telemetry event types** (all sims). Each `Event`/`Observation` is app
  vocabulary; the generic fan-out is `telemetry.Bus`.
- **Distance-shading curves** (pandemonium, nemesis). How brightness falls off
  with distance is art direction — different formulas, different constants.
  Only the pixel op (`paint.Scale`) is shared.
- **Neural networks** (vivarium `neural`, galapagos `nn`). Genuinely different
  nets (recurrent per-agent brains vs. feed-forward NEAT genomes); the shared
  surface is too thin to be worth a common package.

The CLI entrypoint also stays out of crucible: the root command and the
`completion` subcommand (copied verbatim across rubix, gambit, and vivarium
today) are CLI plumbing, not engine code, so each app keeps its own
`internal/cli`. CONVENTIONS.md holds the family to a single shape for them.

Dependencies consumed rather than consolidated: ordinex sorts the hub's
window ids, off the per-frame hot path — render-time sorts keep their
in-place insertion sorts deliberately.

retrievium is deliberately **not** a crucible dependency. Its searchers do
exact-match membership on a sorted slice, and every such list in the engine
(tile kinds, window ids, a handful of shells) is small enough that a linear
scan is as good — a binary-search dependency here would be decoration, not
value. Its honest fits are app-local exact-match lookups, called out in the
gambit and nemesis guides. The one search the engine genuinely wants is
threshold selection — pick an index by cumulative weight — which is a
different search (upper-bound, not exact-match) and lives in
`worldgen.WeightedChoice` on the standard library's `sort.Search`.
