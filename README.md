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
| `worldgen` | BSP dungeon generation, corridors, flood-fill | pandemonium, nemesis |
| `raycast` | Raycasting camera, DDA, billboard projection | pandemonium, nemesis |
| `camera` | 2D pan/zoom camera for top-down views | vivarium |

Only `menu` and `camera` import Ebiten; everything else is plain Go and runs headless.

## Quick start

```go
import (
    "github.com/danielriddell21/crucible/raycast"
    "github.com/danielriddell21/crucible/worldgen"
)

// Carve a connected dungeon into your own tile grid.
rooms := worldgen.Generate(worldgen.NewRNG(seed), myGrid, worldgen.Config{})

// Cast a ray for every screen column.
cam := raycast.NewCamera(playerPos, playerAngle, fov)
for x := range screenW {
    rx, ry := cam.RayDir(x, screenW)
    hit := raycast.Cast(cam.Pos, rx, ry, myGrid.Solid)
    zbuf[x] = hit.Dist
}
```

See each package's godoc for the full surface, and [`docs/provenance.md`](docs/provenance.md) for exactly which files in which repos each package replaces.

## Migrating a family repo

The [`migration/`](migration/) folder holds one guide per repo — rubix, gambit, vivarium, galapagos, hegemony, pandemonium, and nemesis — mapping its current files to crucible packages, call site by call site. ordinex and retrievium need no migration: crucible consumes them as dependencies.

## Why a library, not a framework

Each game keeps its own `internal/gui`, `ebiten.Game`, renderer, and simulation — crucible supplies the shared machinery behind that seam, not a scene graph. The family's [CONVENTIONS.md](CONVENTIONS.md) still governs the shape of each app; crucible is the code those conventions kept describing.
