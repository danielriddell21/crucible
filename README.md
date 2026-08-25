# crucible

*n.* a vessel in which substances are melted down and fused at great heat. Also: the engine the rest of the family melted into.

The shared Ebitengine app/game engine behind the tool family — the GUI panels, world generation, title screens, and window plumbing that rubix, gambit, vivarium, galapagos, hegemony, pandemonium, and nemesis were each carrying copies of.

[![Go Reference](https://pkg.go.dev/badge/github.com/danielriddell21/crucible.svg)](https://pkg.go.dev/github.com/danielriddell21/crucible)
[![CI](https://github.com/danielriddell21/crucible/actions/workflows/ci.yaml/badge.svg)](https://github.com/danielriddell21/crucible/actions/workflows/ci.yaml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danielriddell21_crucible2&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=danielriddell21_crucible2)
[![Go 1.26](https://img.shields.io/badge/go-1.26-blue)](https://go.dev)
[![MIT License](https://img.shields.io/badge/licence-MIT-green)](LICENSE)

## Install

```sh
go get github.com/danielriddell21/crucible@latest
```

## Packages

One package per engine concern, flat at the module root. App-specific vocabularies (messages, events, cues, tiles, textures) stay in each app; packages that carry them are generic.

| Package | What it is | Consolidated from |
|---|---|---|
| `geom` | 2D/3D vectors, grid coords, rects, clamps, angle folding, easing | vivarium, pandemonium, nemesis, hegemony, autobahn |
| `hud` | Timed status-line overlay | pandemonium, nemesis (identical copies) |
| `canvas` | Software RGBA text/menu surface | nemesis, pandemonium |
| `menu` | Title/pause/settings menu model, display-free | nemesis, pandemonium, autobahn |
| `menu/ebiteninput` | The conventional menu bindings, read from Ebiten | nemesis, pandemonium |
| `keymap` | Display-free control-hint bar layout | the family's control bars |
| `window` | Shared Ebiten window setup and resizing policy | the family's front-ends |
| `store` | JSON settings/records under the user config dir | nemesis, pandemonium |
| `status` | Cue → timed HUD line pipeline | pandemonium |
| `narrate` | narrata-backed rewording of status lines | pandemonium |
| `telemetry` | Generic observer bus with bounded recent feed | pandemonium, nemesis |
| `synth` | Procedural PCM sound effects, panning, cue shapes | pandemonium, nemesis |
| `record` | Frame capture to GIF, MP4 and PNG | rubix, gambit, galapagos, pandemonium |
| `demo` | Headless documentation-media toolkit — clips, montages, palette ramps | the family's `tools/demogen` |
| `hub` | Multi-window leader/child coordination | rubix, hegemony, nemesis |
| `worldgen` | BSP dungeons, corridors, flood-fill, weighted choice, and seeded grid lattices | pandemonium, nemesis, autobahn |
| `level` | Shared world model + generation pipeline: tiles, heights, half walls, lifts, doors, vents, themes, sky | pandemonium, nemesis |
| `raycast` | Raycasting camera, DDA, billboards, height-aware column walker | pandemonium, nemesis |
| `view` | Display-free 2D pan/zoom camera: world↔screen, follow, fit-to-bounds | galapagos |
| `camera` | Ebiten draw transform layered over a 2D view | vivarium |
| `rng` | Deterministic seeded sub-streams (PCG) | galapagos, hegemony |
| `ring` | Fixed-capacity rolling-history buffer | galapagos, vivarium |
| `paint` | Colour brightness scaling and full-frame framebuffer blend | pandemonium, nemesis, autobahn |
| `spatial` | Uniform grid for radius queries, open-plane or toroidal | vivarium, autobahn |
| `netplay` | Two-player host/join sessions over gob, with a lobby-facing session | autobahn |
| `pinhole` | Perspective camera: world→image projection and ground ranging | autobahn |

Only `menu/ebiteninput`, `camera` and `window` import Ebiten; everything else is plain Go and runs headless. `menu` itself is display-free, which is how a raylib front-end shares the family's menus.

## Quick start

```go
import (
    "github.com/danielriddell21/crucible/level"
    "github.com/danielriddell21/crucible/raycast"
)

// Generate a furnished level: rooms, corridors, heights, half walls,
// doors, vents, themes, sky — retried until the exit is reachable.
l, rooms, err := level.Generate(level.GenerateConfig{Width: 64, Height: 48, Seed: seed},
    []level.Pass{
        func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
            level.AssignHeights(l, rng, rooms, level.HeightsConfig{})
            level.PlaceDoors(l, rng, level.DoorConfig{})
            level.CarveVents(l, rooms, level.VentConfig{})
        },
    }, nil)

// Cast a ray for every screen column; WalkColumn handles variable
// heights and half walls, painting through your renderer.
cam := raycast.NewCamera(playerPos, playerAngle, fov)
for x := range screenW {
    res := raycast.WalkColumn(cam, screenW, screenH, x, eyeZ, l, myPainter)
    zbuf[x] = res.Depth
}
```

See each package's godoc for the full surface, and the [Provenance](https://github.com/danielriddell21/crucible/wiki/Provenance) wiki page for exactly which files in which repos each package replaces.

## The family, migrated

All seven apps — rubix, gambit, vivarium, galapagos, hegemony, pandemonium, and nemesis — now build on crucible. The [Provenance](https://github.com/danielriddell21/crucible/wiki/Provenance) wiki page maps every package to the files it replaced, and records what was deliberately left app-side and why. ordinex needs no migration — crucible consumes it as a dependency (the `hub` sorts window ids with it) — and retrievium stays app-side, its sorted-slice search a linear scan's-worth of value away from being worth a dependency.

## Why a library, not a framework

Each game keeps its own `internal/gui`, `ebiten.Game`, renderer, and simulation — crucible supplies the shared machinery behind that seam, not a scene graph. The family's [CONVENTIONS.md](CONVENTIONS.md) still governs the shape of each app; crucible is the code those conventions kept describing.
