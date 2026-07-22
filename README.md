# crucible

*n.* a vessel in which substances are melted down and fused at great heat. Also: the engine the rest of the family melted into.

The shared Ebitengine app/game engine behind the tool family — the GUI panels, world generation, title screens, and window plumbing that rubix, gambit, vivarium, galapagos, hegemony, pandemonium, and nemesis were each carrying copies of.

[![Go Reference](https://pkg.go.dev/badge/github.com/danielriddell21/crucible.svg)](https://pkg.go.dev/github.com/danielriddell21/crucible)
[![CI](https://github.com/danielriddell21/crucible/actions/workflows/ci.yaml/badge.svg)](https://github.com/danielriddell21/crucible/actions/workflows/ci.yaml)
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
| `geom` | Vectors, grid coords, rects, clamps | vivarium, pandemonium, nemesis, hegemony |
| `hud` | Timed status-line overlay | pandemonium, nemesis (identical copies) |
| `canvas` | Software RGBA text/menu surface | nemesis, pandemonium |
| `menu` | Title/pause/settings menu model + input polling | nemesis, pandemonium |
| `store` | JSON settings/records under the user config dir | nemesis, pandemonium |
| `status` | Cue → timed HUD line pipeline | pandemonium |
| `narrate` | narrata-backed rewording of status lines | pandemonium |
| `telemetry` | Generic observer bus with bounded recent feed | pandemonium, nemesis |
| `synth` | Procedural PCM sound effects, panning, cue shapes | pandemonium, nemesis |
| `record` | Demo GIF recorder and PNG screenshots | rubix, gambit, galapagos, pandemonium |
| `hub` | Multi-window leader/child coordination | rubix, hegemony, nemesis |
| `worldgen` | BSP dungeon generation, corridors, flood-fill, weighted choice | pandemonium, nemesis |
| `level` | Shared world model + generation pipeline: tiles, heights, half walls, lifts, doors, vents, themes, sky | pandemonium, nemesis |
| `raycast` | Raycasting camera, DDA, billboards, height-aware column walker | pandemonium, nemesis |
| `view` | Display-free 2D pan/zoom camera: world↔screen, follow, fit-to-bounds | galapagos |
| `camera` | Ebiten draw transform layered over a 2D view | vivarium |
| `rng` | Deterministic seeded sub-streams (PCG) | galapagos, hegemony |
| `ring` | Fixed-capacity rolling-history buffer | galapagos, vivarium |
| `paint` | Colour brightness scaling and full-frame framebuffer blend | pandemonium, nemesis |

Only `menu` and `camera` import Ebiten; everything else is plain Go and runs headless.

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

See each package's godoc for the full surface, and [`docs/provenance.md`](docs/provenance.md) for exactly which files in which repos each package replaces.

## Migrating a family repo

The [`migration/`](migration/) folder holds one guide per repo — rubix, gambit, vivarium, galapagos, hegemony, pandemonium, and nemesis — mapping its current files to crucible packages, call site by call site. ordinex needs no migration: crucible consumes it as a dependency (the `hub` sorts window ids with it). retrievium stays app-side — the guides note where its sorted-slice search fits each app.

## Why a library, not a framework

Each game keeps its own `internal/gui`, `ebiten.Game`, renderer, and simulation — crucible supplies the shared machinery behind that seam, not a scene graph. The family's [CONVENTIONS.md](CONVENTIONS.md) still governs the shape of each app; crucible is the code those conventions kept describing.
