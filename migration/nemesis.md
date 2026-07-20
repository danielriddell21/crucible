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

## 8. `internal/world/bsp.go` + `connectivity.go` → `crucible/worldgen`

As pandemonium §6, but simpler: nemesis has no height rule, so every flood
call passes a `nil` step gate. `Room` is `geom.Rect` (same field layout),
`Coord` is `geom.Coord`, the local `rng` is `worldgen.RNG` with identical
seeding — fixed-seed decks regenerate identically. `vents.go`,
`objectives.go`, `lockers.go`, doors, and spawn placement stay, now calling
`worldgen.FloodDist`/`Reachable`/`StepsBetween`.

## 9. `internal/render` → `crucible/raycast` (geometry only)

- `castRay` in `walls.go` → `raycast.Cast` for plain columns; the sliding
  `doorColumn` logic keeps its own loop on
  `raycast.BoundaryDist`/`BoundaryWallX` (exported for exactly this).
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
