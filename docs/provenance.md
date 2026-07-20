# Provenance

Where each crucible package came from: the files that were reviewed and
consolidated, and what deliberately stayed behind in each repo. Pandemonium
was reviewed at its `feat/audio-cues` branch; every other repo at `trunk`.

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
| `worldgen` | pandemonium `internal/world/bsp.go` + `connectivity.go`; nemesis `internal/world/bsp.go` + `connectivity.go` | tile enums, levels, doors, items, heights (pandemonium's step rule plugs in via the flood `step` gate) |
| `raycast` | nemesis `internal/render/walls.go` (DDA) + `sprites.go` (projection); pandemonium `internal/render/camera.go` + `columns.go` | texturing, shading, framebuffers, door/height column logic (built on the exported boundary helpers) |
| `camera` | vivarium `internal/gui/camera.go` | input wiring |

Dependencies consumed rather than consolidated: ordinex sorts the hub's
window ids. ordinex is used
only off the per-frame hot paths — render-time sorts keep their in-place
insertion sorts deliberately.
