# nemesis → crucible

nemesis shares the raycaster-family machinery with pandemonium and has the
family's third copy of the multi-window hub. Its alien AI, director,
learning, senses, vents/lockers, textures, and sound design stay put.

## 1. `internal/hud/overlay.go` → `crucible/hud`

Byte-identical to crucible's copy. Delete the package, rewrite imports.

## 2. `internal/gui/canvas.go` → `crucible/canvas`

The software menu surface is crucible's `canvas` with exported names:
`newCanvas(w, h)` → `canvas.New(w, h)`, `c.pixels()` → `c.Pixels()`,
`fill`/`dimFrom`/`rect`/`text`/`textCentered` keep their behaviour
(including the drop shadow). `glyphWidth` is `canvas.GlyphWidth`.

## 3. `internal/gui/menu.go` + `menuflow.go` → `crucible/menu`

`menuItem`/`menu` map to `menu.Item`/`menu.Menu` (exported fields:
`label` → `Label`, `action` → `Action`, `adjust` → `Adjust`, `sel` → `Sel`).
`m.update()` becomes `m.Update(menu.Poll())` — the key bindings are
identical — and `menuSound` maps to `menu.Sound`
(`soundNone/Move/Select` → `SoundNone/Move/Select`). `m.draw(c)` becomes
`m.Draw(c, theme)`; the local colour vars become a `menu.Theme` (nemesis's
palette is crucible's `menu.DefaultTheme()`). `buildMenus`,
`buildSettingsMenu`, `openSettings`, `leaveSettings` stay — they are the
game's menus; `bar`/`onOff` helpers come from `menu.Bar`/`menu.OnOff`.

## 4. `internal/gui/settings.go` + `records.go` → `crucible/store`

Keep the `Settings`/`Records` structs, `clamped()`, `summary()`, and the
mutators; replace path/load/save plumbing with
`store.Path("nemesis", "settings.json")`, `store.Load(path, defaultSettings())`,
`store.Save(path, s)`. The local `clamp01`/`clampRange` become
`geom.Clamp01`/`geom.Clamp`.

## 5. `internal/cli/coord.go` + `internal/gui/link.go` → `crucible/hub`

Same swap as rubix (see rubix.md §2) with nemesis's `Msg`/`StateMsg` as the
type parameter. The local `trySend` in `link.go` is `hub.TrySend`. Keep
`Msg` and `StateMsg` in `internal/gui`.

## 6. `internal/audio/synth.go` → `crucible/synth`

Keep `Cue`, `CueFor`, `Synth()`, `Ambient(band)`, `MenaceBand` — the sound
design. The primitives move:

- `renderPCM`/`env` → `synth.Render`/`synth.Env`; format constants from
  `synth` (`BytesPerFrame` is exported).
- `Pan`/`Panned` → `synth.Pan`/`synth.Panned` (moved verbatim).
- Cue bodies that match a shared shape can use it (`synth.Blip` for the
  menu ticks, `synth.Rumble` for the door, `synth.Slide` for the death
  groan — pass the current PCG seeds to keep noise layers
  sample-identical); the station-specific ones (ping, creak, hiss,
  screech, decoy) stay local on top of `Render`/`Env`/`Noise`/`Sine`.

## 7. `internal/telemetry` → `crucible/telemetry` (mechanics only)

Keep `Event`, `FromObservation`, `Line()`, and `roman` — the feed wording.
The bus becomes `telemetry.Bus[Event]`: nil-subscriber filtering and the
64-deep recent feed are `NewBus(...)` +
`Configure(WithFeedDepth[Event](64))`, and the `ObsStep` mute becomes
`WithFilter`. `Recent()` keeps its semantics.

Optional retrievium fit: `Event.Kind()` scans kind names linearly — with a
sorted `[]string` of kind names, `retrievium.BinarySearcher[string]` does
the lookup (cold path, called from tooling).

This is retrievium's honest home. Its searchers do *exact-match* membership
("is `want` in this sorted slice, and where"), which is exactly what a
name→kind lookup is. That's a genuine but app-local fit, so retrievium
stays a nemesis dependency and is deliberately **not** pulled in through
crucible — the engine's searchable lists are all tiny enough that a linear
scan is as good, so wiring a binary-search dependency into the engine would
be decoration, not value. (Threshold-style "pick by cumulative weight"
selection, which the engine *does* want, is a different search and lives in
crucible as `worldgen.WeightedChoice`, built on stdlib `sort.Search`.)

## 8. `internal/world` → `crucible/level` (+ `worldgen`)

`TileType` → `level.Tile` with two renames in the engine vocabulary:
`TileConsole` → `level.TileSwitch` and `TileLocker` → `level.TileCover`
(same walkability, same runes). The local `Level` becomes `level.Level`;
`Room` is `geom.Rect`, `Coord` is `geom.Coord`, the local `rng` is
`worldgen.RNG` with identical seeding.

| Local | Crucible |
|---|---|
| `generate.go` attempt loop + `placeSpawnAndExit` | `level.Generate(cfg, passes, validate)` — keep the `len(l.VentMouths) >= 2` check in the validator |
| bsp/corridors/stubs | `worldgen.Generate` (run by the pipeline) |
| `doors.go` (`placeDoors`) | `level.PlaceDoors(l, rng, level.DoorConfig{})` |
| `vents.go` (`carveVents`) | `level.CarveVents(l, rooms, level.VentConfig{})` — **improved**, see below |
| `connectivity.go` | `worldgen.FloodDist`/`Reachable`/`StepsBetween` with a `nil` step gate |

`CarveVents` is better than the local copy in three ways: mouths open at
the centre of the wall span a room shares with the mass (the old
perimeter scan biased them into top-left corners); each mouth joins the
network by the cheapest path to the nearest already-carved tunnel, so
networks branch like trees instead of snaking room to room in room-index
order; and a configurable depth bias steers tunnels away from wall faces
so they stay hidden from the rooms they pass. Same determinism guarantee;
vent layouts for a given seed will differ from the old algorithm's.

Stays app-side as `level.Pass` values: consoles/lockers placement (now
setting `TileSwitch`/`TileCover`), objectives, light moods and `Flicker`
(keep the flicker slice beside the level), runtime door slide state.
Bonus from pandemonium, free to adopt: `level.AssignHeights`,
`level.PlaceLowWalls`, `level.PlaceLiftLedge` + `level.LiftHeight`,
`level.AssignThemes`, `level.AssignSky`, and `(*level.Level).StepOK` if
decks ever gain height.

## 9. `internal/render` → `crucible/raycast`

- `castRay` in `walls.go` → `raycast.Cast` for plain columns; the sliding
  `doorColumn` logic keeps its own loop on
  `raycast.BoundaryDist`/`BoundaryWallX` (exported for exactly this).
- Alternatively adopt `raycast.WalkColumn` wholesale (a `level.Level`
  satisfies `raycast.Heights` directly): nemesis then renders heights,
  half walls, and per-cell floors/ceilings the same way pandemonium does,
  via a `raycast.ColumnPainter` that keeps nemesis's textures and palette.
- The camera struct in `renderer.go` → `raycast.Camera`
  (`NewCamera(pos, angle, fov)`, `RayDir`).
- `drawBillboard`'s projection → `raycast.Camera.Project`;
  `sortByDepth` → `raycast.SortFarToNear(boards, depth)`.
- Textures, palette, shading, HUD drawing: stay.

## 10. Screenshot/records capture

If the visualiser grows GIF capture to match rubix, use `crucible/record`
rather than porting rubix's file again.

## Order of work

1. `hud`, `canvas`, `store` (mechanical).
2. `menu` (verify title/pause/settings by hand).
3. `synth`, `telemetry`.
4. `worldgen` (fixed-seed regression), then `raycast`.
5. `hub` last; verify the visualiser's multi-window keys.
