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

## 5. `internal/cli/coord.go` → `crucible/hub`

The whole local hub (`newHub`, `addParticipant`, `broadcastExcept`, `drop`,
`spawnChild`, `childLink`) is `hub.Hub[gui.Msg]`. nemesis spawns exactly one
child — the visualiser — at startup, so `lead()` drives the primitives
directly like hegemony rather than going through `hub.RunLeader`
(`hub.New` → `AddParticipant` the game as id 0 → `go Run()` → a goroutine
draining `leaderOut` into `Inject(0, …)` → `SpawnChild()`). `runChild` is
`hub.RunChild`. The routing policy is a small `route(gui.Msg) hub.Route`:
`hello` → `RouteState` (the shared level a late visualiser needs, cached and
replayed — this replaces the local `last`/replay), `state`/`events` →
`RouteBroadcast`, anything else → `RouteNone`.

Keep `Msg`/`StateMsg` and the game's own `trySend` in `internal/gui`
(`link.go` is untouched). Because `hub` imports `ordinex/v2`, `go mod tidy`
adds that indirect entry; and `hub.RunChild` returns an external error, so
wrap it (see the wrapcheck note in the README).

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

## 8. `internal/world` → `crucible/level` (+ `geom`, `worldgen`)

nemesis fully adopts the engine's world model. The regeneration is
deliberate and accepted: the improved vents and the full-area BSP change
every fixed-seed layout, so the tests are invariant-based (connectivity,
console reachability, ≥2 vent mouths, border intact, determinism) rather
than golden grids.

The vocabulary moves to `level.Tile`. nemesis keeps its own domain names as
thin aliases — exactly the engine's "games name tiles their own way"
pattern:

```go
type Tile = level.Tile
const (
    TileConsole = level.TileSwitch // a wall-mounted interactable
    TileLocker  = level.TileCover  // a hiding spot
    // Floor/Wall/Door/Vent/Spawn/Exit map straight across
)
```

`Coord`/`Room` are `geom.Coord`/`geom.Rect`. The `world.Level` becomes an
aggregate that **embeds `*level.Level`** (promoting `At`, `Solid`, `LightAt`,
`Spawn`, `Exit`, `VentMouths`, `W`/`H`, …) and adds nemesis's gameplay
markers — the engine rule keeps items and markers app-side:

```go
type Level struct {
    *level.Level
    Rooms    []Room
    Consoles []Coord
    Lockers  []Coord
    Flicker  []bool
}
```

Generation is `level.Generate(cfg, passes, validate)`:

| Local | Crucible |
|---|---|
| bsp/corridors/stubs, `placeSpawnAndExit` | run by the pipeline (`worldgen.Generate` + `PlaceSpawnExit`) |
| `carveVents` | `level.CarveVents` — the improved tree-branching version |
| `placeDoors` | `level.PlaceDoors` (byte-identical to the old copy) |
| `connectivity.go` | `worldgen.FloodDist`/`Reachable` |

The console, locker, and light passes stay app-side as `level.Pass` closures
that set `TileSwitch`/`TileCover` and record the marker slices onto the
`Level` aggregate; `validate` keeps nemesis's viability rule (exit reachable,
console count and faces reachable, ≥2 vent mouths). Two gotchas: `level.New`
starts every cell fully lit, so `assignLight` clears the light field before
laying its moods; and door-slide runtime state stays in `sim.World`, whose
`Solid` still blocks closed doors (the engine's `Level.Solid` treats a door
as walkable). The remaining `level` features (`AssignHeights`,
`PlaceLowWalls`, lifts, themes, sky) are there for later if nemesis grows
heights.

## 9. `internal/render` → `crucible/raycast`

The camera and projection come from the engine: the local `camera`/
`newCamera` become `raycast.NewCamera`/`Camera.RayDir`, `drawBillboard`'s
projection is `Camera.Project`, and `sortByDepth` is `SortFarToNear`. A
`rayHit` is a `raycast.Hit`.

The one thing kept local is the **ray walk itself**: nemesis animates
sliding doors by letting a ray pass through the retracted fraction of a
half-open door (`wallX < slide`), which `raycast.WalkColumn`'s binary
`Solid` predicate can't express without teaching the engine about doors. So
`castRay` keeps its own DDA loop built on the exported
`raycast.BoundaryDist`/`BoundaryWallX` — which exist for exactly this
sliding-door case — and `sim.World.Solid` remains the door-aware blocker.
Column distances, projection, and sort are byte-identical to the old code.
`WalkColumn` and per-cell heights are left for if nemesis ever renders a
heightfield; textures, palette, shading, and HUD drawing stay local.

## 10. Screenshot/records capture

If the visualiser grows GIF capture to match rubix, use `crucible/record`
rather than porting rubix's file again.

## Order of work

1. `hud`, `canvas`, `store` (mechanical).
2. `menu` (verify title/pause/settings by hand).
3. `synth`, `telemetry`.
4. `hub`; verify the visualiser's multi-window keys.
5. `geom` aliases and `worldgen` flood-fill first (output-neutral), then the
   full `level` adoption and `raycast` — regeneration lands here, so lean on
   the invariant tests and a `demogen` run to eyeball the maps and the 3D
   view.

Done on `refactor/adopt-crucible`, one commit per package. Only the sliding-
door ray walk stays local (§9); everything else moves to the engine.
