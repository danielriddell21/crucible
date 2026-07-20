# pandemonium → crucible

Reviewed at **feat/audio-cues** (the current head, ahead of trunk). This is
the largest migration: pandemonium contributed the status/narrate pipeline
and shares the raycaster-family machinery with nemesis. Its sim, world
tiles/heights/gates, textures, and sound design stay put.

## 1. `internal/hud/overlay.go` → `crucible/hud`

Byte-identical to crucible's copy. Delete the package, rewrite imports to
`crucible/hud`. No call sites change.

## 2. `internal/status` → `crucible/status` (contract only)

Keep the cue vocabulary (`CueKind`, `Cue`), the band policy, and the
scripted tables in `internal/status` — they are the game's voice. Replace
the local plumbing types with crucible's:

- `status.Line` → `crucible/status.Line` (same fields).
- `type Source interface { Request(...) }` → `crucible/status.Source[Cue]`.
- `NewTableSource()` now returns a `crucible/status.Source[Cue]`
  (`status.Func[Cue]` adapts the existing function directly).
- The reporter's `emit` closure can be `status.Emit(overlay)`.

## 3. `internal/phrasing` → `crucible/narrate`

`phrasing.Source` is `narrate.Source[status.Cue]`; keep `personas.json`
and the `eventFor`/`dataFor` mappings — they become the two functions
passed to `narrate.New`:

```go
src, err := narrate.New(narrate.Config{
    Personas: personasJSON, Persona: "house",
    MaxWords: 14, Timeout: 2 * time.Second, MaxConcurrent: 2,
}, status.NewTableSource(), eventFor, dataFor)
```

The temp-file dance, the Notice-only rewrite rule, and the
scripted-fallback behaviour are all inside `narrate`. Delete
`writeTemp`, `generate`, and the `Request` method.

## 4. `internal/telemetry` → `crucible/telemetry` (mechanics only)

Keep `PlayerEvent`, `PathSummary`, `RunProfile`, the path aggregation, and
`markerFor` — that is pandemonium's analytics. What moves is the bus
mechanics: fan-out and subscriber management become
`telemetry.Bus[PlayerEvent]` (nil subscribers are now filtered for free).
The three-method `Subscriber` interface can stay app-side; adapt it as one
`OnEvent` subscriber plus direct calls, or keep the local Bus as a thin
aggregator that embeds `telemetry.Bus[PlayerEvent]` for dispatch.

## 5. `internal/audio/synth.go` → `crucible/synth`

Keep `Cue`, `CueFor`, `Synth()`, `Ambient()`, `Music()` — the sound design.
Replace the primitives underneath:

| Local | Crucible |
|---|---|
| `SampleRate`, `ChannelCount`, `BitDepthInBytes`, `bytesPerFrame` | `synth.SampleRate` etc. (`BytesPerFrame` is exported) |
| `renderPCM(dur, gen)` | `synth.Render(dur, gen)` |
| `env(t, decay)` | `synth.Env(t, decay)` |
| `synthMenu` | `synth.Blip(880, 0.05, 60)` |
| `synthPickup` | `synth.TwoTone(660, 990, 0.08, 0.18, 22)` |
| `synthSecret` | `synth.Arpeggio([]float64{523, 659, 784}, 0.15, 0.45, 10)` |
| `synthDeath` | `synth.Slide(300, -360, 60, 0.6, 4)` |
| `synthDoor` | `synth.Rumble(90, 40, 0.40, 5, 6)` |
| `synthHit` | `synth.Thud(150, 0.10, 55, 3, 4)` |
| `synthFire`, `Music`, `Ambient` | keep local, built on `synth.Render`/`Env`/`Noise`/`Sine` |

Rendered bytes are identical for the pure-tone cues; the noise-based ones
stay sample-identical only if the same PCG seeds are passed (shown above).

## 6. `internal/world` → `crucible/level` (+ `worldgen`)

Most of `internal/world` **is** crucible now — pandemonium contributed the
model. `TileType` → `level.Tile` (`TileSwitch` keeps its name and meaning),
and the local `Level` becomes `level.Level` with the same field shapes
(`FloorH`, `CeilH`, `WallTop` → `WallTopH`, `Light`, `Sky`, `Theme`,
`Lifts`); `Coord` is `geom.Coord`, `rect` is `geom.Rect`, the local `rng`
is `worldgen.RNG` (identical seeding).

| Local | Crucible |
|---|---|
| `generate.go` `Generate` (attempt loop, sub-seed derivation, validation) | `level.Generate(cfg, passes, validate)` — same sub-seed constant; put `keysReachable` in the validator |
| `placeSpawnAndExit` | `level.PlaceSpawnExit` (run automatically by the pipeline) |
| bsp/corridors/stubs | `worldgen.Generate` (run automatically) |
| `heights.go` (`assignHeights`) | `level.AssignHeights(l, rng, rooms, level.HeightsConfig{})` |
| `placeLiftLedge` + sim `liftHeight` | `level.PlaceLiftLedge` (takes an `avoid` predicate for items/secrets and returns the ledge for the reward drop) + `level.LiftHeight(lift, t, dwell, travel)` |
| `lowwall.go` | `level.PlaceLowWalls` |
| `connectivity.go` (`stepOK`, flood) | `(*level.Level).StepOK` with `level.DefaultMaxStep`/`DefaultMinHeadroom` as the `step` gate of `worldgen.FloodDist` |
| `theme.go` | `level.AssignThemes(l, rng, rooms, 3)` |
| `sky.go` | `level.AssignSky(l, rng, rooms, level.SkyConfig{})` |

Stays app-side, wrapped as `level.Pass` values in the pipeline: markers
(`annotate.go`), gates/keys, items, barrels, hazards, switches, arenas,
and `light.go`'s mood policy. Bonus from nemesis: `level.PlaceDoors` and
`level.CarveVents` are now available if pandemonium ever wants sliding
doors or crawl spaces.

## 7. `internal/render` → `crucible/raycast`

- `camera.go` → `raycast.Camera` (`NewCamera(pos, angle, fov)`,
  `RayDir(x, w)`); note crucible's camera also carries `Pos`.
- `columns.go`'s `drawColumn` → `raycast.WalkColumn`: the DDA, the
  shrinking visible window, departed floor/ceiling fills, step and
  ceiling-drop faces, half-wall see-over, and the low-wall occlusion
  triple (`loZ`/`loH`/`loRow` → `ColumnResult`) all move to the engine.
  What remains is a `raycast.ColumnPainter` implementation: `WallSpan`
  keeps `drawWallSpan` (texture pick via `Face.Cell`/`Face.From` theme +
  light, `Face.WallX`/`Flip` for the texture column), `FloorSpan` keeps
  `fillFloorSpan` (hazard textures by cell), `CeilSpan` picks stone or
  `fillSkySpan` via `SkyAt`.
- Sprite placement math → `raycast.Camera.Project`; the per-frame depth
  ordering in `sprites.go:71` can use `raycast.SortFarToNear`.
- Textures, shading, gloom, automap, statusbar, viewmodel: stay.

## 8. `internal/gui` housekeeping

- `settings.go`/`records.go`: keep the structs, clamps, and summaries;
  replace the load/save/path plumbing with `store.Path("pandemonium", ...)`,
  `store.Load(path, defaults)`, `store.Save(path, v)`. Load semantics
  match (missing/corrupt → defaults); saves keep the trailing newline.
- `screenshot.go`: the PNG encode becomes
  `record.SavePNG(name, record.FromRGBA(fb, w, h))`; the HUD toggling
  around it stays.
- `menu.go`: rebuild on `crucible/menu` (`menu.Menu`, `menu.Item`,
  `menu.Poll`, `menu.Bar`, `menu.OnOff`), drawing via `crucible/canvas`
  like nemesis does today.

## 9. Not in this migration

- `internal/sim`, `internal/world` beyond §6 (heights, gates, items,
  themes), `internal/status`'s wording, `tools/demogen`.
- pandemonium has no multi-window mode, so `hub` is unused.

## Order of work

1. `hud` (mechanical), then `store`, then `record`.
2. `synth` (verify cues by ear or by hashing the PCM).
3. `telemetry`, `status`, `narrate` (one commit each; the phrasing tests
   port almost verbatim).
4. `worldgen` (fixed-seed map hashes make a good regression test).
5. `raycast`, `menu`/`canvas` last — they touch the most files.
