# vivarium → crucible

vivarium contributed `geom` and `camera` to crucible; the simulation,
neural nets, and analytics stay put.

## 1. `internal/geom` → `crucible/geom`

Crucible's `geom.Vec2` is a superset of vivarium's (`Add`, `Sub`, `Scale`,
`Len`, `Normalize`, `Angle`, `FromAngle`, `WrapTo`, `ShortestDelta`,
`ToroidalDist` all keep their names and semantics; `Dot`, `Dist`, `DistSq`
are new). Delete `internal/geom` and rewrite imports to
`github.com/danielriddell21/crucible/geom`. No call sites change.

## 2. `internal/gui/camera.go` → `crucible/camera`

The pan/zoom camera is crucible's `camera` package with the fields exported:

| Local | Crucible |
|---|---|
| `newCamera()` | `camera.New()` |
| `c.geoM()` | `c.GeoM()` |
| `c.screenToWorld(sx, sy)` | `c.ScreenToWorld(sx, sy)` |
| `c.zoomAt(f, sx, sy, w, h)` | `c.ZoomAt(f, sx, sy, w, h)` |
| `c.pan(dx, dy, w, h)` | `c.Pan(dx, dy, w, h)` |

The zoom bounds (1..12) moved from the hard-coded clamp into
`MinZoom`/`MaxZoom` fields; `camera.New()` sets the same 1..12. The local
`clamp` helper's other users switch to `geom.Clamp`.

Note `crucible/camera` imports Ebiten (for `GeoM`), so it may only be used
from `internal/gui`'s `ebiten`-tagged files — which is where the local
camera already lives.

## 3. `internal/gui/phylogeny.go` — ordinex fit

`phylogeny.go:39,84,86` sorts plain `[]int` species ids with `sort.Ints`
outside the draw hot path — swap for
`ordinex.MergeSorter[int]{}.Sort(ids)` (it returns a copy; reassign).
`lineage.go:57` is a struct `sort.Slice`; keep it.

## 4. Stays put

- `internal/cli` — the command tree, root, and `completion.go` all stay;
  crucible is engine code, not CLI plumbing.

- `internal/sim`, `internal/neural`, `internal/analytics` — vivarium's
  domain.
- The `gui.Run`/`Available` seam and `//go:build ebiten` stubs — unchanged,
  per CONVENTIONS.md.

## Order of work

1. `geom` (mechanical import rewrite).
2. `camera`.
3. Phylogeny ordinex swap.
