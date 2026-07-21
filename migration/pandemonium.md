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

## 4. `internal/telemetry` → **not adopted**

pandemonium's `telemetry.Bus` is a multi-channel domain aggregator: one
`Observe(sim.Observation)` fans work out to a three-method `Subscriber`
(`OnEvent`/`OnPathSummary`/`OnRunProfile`) while it also builds `PathSummary`
and `RunProfile` from the raw stream. crucible's `telemetry.Bus[E]` is a
single-channel fan-out (`Publish`/`Subscribe`/`Recent`). Routing only the
`PlayerEvent` channel through it would need three separate buses and leave
the aggregation behind — more code, not less — so telemetry stays local.
(This is the honest counterpart to the clean fits: adopt where the engine
genuinely subsumes the app code, not where the shapes merely rhyme.)

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

The app-specific passes stay, wrapped as `level.Pass` closures over a
`world.Level` aggregate that embeds `*level.Level` and adds the gameplay
layer (markers, items, locks, secrets, barrels, hazards, switches). Each
pass resets the aggregate to the freshly-dug level and mutates it in order
(exit switch, `annotate`, key gate, items, barrels, lift reward, hazards,
light); `Solid` is overridden so a closed door still blocks. Bonus from
nemesis: `level.PlaceDoors`/`level.CarveVents` are there if pandemonium
ever wants sliding doors or crawl spaces.

**Regeneration is expected.** `worldgen.Generate` roots its BSP at the full
area where the old code inset by one cell, so every fixed-seed map changes
(the one-cell border survives via `RoomPad`). The world tests are
invariant-based — connectivity, key reachability, single-step heights,
dais, sky, border intact — not golden grids. Two knock-on adjustments in
the same commit: barrels now avoid one-wide corridor cells (a solid barrel
must not wall off a mandatory path — good for the game, and it keeps the QA
roamer from jamming), and `sim/bot`'s `TestRoamerCompletesLevels` samples
many seeds for a stable majority rather than a small fixed count.

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
  The `Heights` implementation reads the sim's **dynamic** floors/ceilings
  (`World.FloorAt`/`CeilAt`) so lifts render at their current height.
- Sprites adopt `raycast.SortFarToNear` for the depth order and
  `raycast.Camera` for the projection vectors, but keep their own inline
  projection: pandemonium anchors each billboard on its own world height
  (grounded vs floating), which `Camera.Project`'s fixed ground line does
  not express.
- Textures, shading, gloom, automap, statusbar, viewmodel: stay. The render
  tests (frame/columns/sky/lowwall) confirm the column output is unchanged.

## 8. `internal/gui` housekeeping

- `settings.go`/`records.go`: keep the structs, clamps, and summaries;
  replace the load/save/path plumbing with `store.Path("pandemonium", ...)`,
  `store.Load(path, defaults)`, `store.Save(path, v)`. Load semantics
  match (missing/corrupt → defaults); saves keep the trailing newline.
- `screenshot.go`: the PNG encode becomes
  `record.SavePNG(name, record.FromRGBA(fb, w, h))`; the HUD toggling
  around it stays.
- `menu.go`: the `menuModel`/`menuEntry` state machine and `render.Menu`
  become `crucible/menu` (`menu.Menu`/`menu.Item`, `Menu.Update` fed the
  local `nav` converted to a `menu.Input`) drawn onto a `crucible/canvas`
  and blitted, like nemesis. Settings rows bake their value into the label
  in a padded column (`adjustRow`); `onOff`/`percent`/`times`/`degrees` stay
  as the value formatters. This **changes the menu's look** to the family's
  canvas font and theme — accepted, in keeping with full adoption.

## 9. Not in this migration

- `internal/sim`, `internal/status`'s wording, `internal/telemetry` (§4),
  `tools/demogen`.
- pandemonium has no multi-window mode, so `hub` is unused.

## Order of work

1. `hud` (mechanical), then `store`, then `record`.
2. `synth` (verify cues by hashing the PCM — the cue map is byte-identical).
3. `status`, `narrate` (one commit; the phrasing tests port almost verbatim).
   `telemetry` is deliberately left local (§4).
4. `level` (invariant tests + a `demogen`/still to eyeball the maps and the
   3D view — regeneration lands here).
5. `raycast`, then `menu`/`canvas` — they touch the most files.

Done on `refactor/adopt-crucible` (off `feat/audio-cues`), one commit per
package.
