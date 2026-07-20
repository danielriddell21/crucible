# galapagos → crucible

galapagos is the family's RL/evolution harness and already imports rubix
and gambit for their domain logic — that stays. Its crucible migration is
the recorder and (eventually) the hub, once its GUI grows multi-window
support to match rubix. `internal/cli` (including the entrypoint and
completion) stays put.

## 1. `internal/gui/record.go` → `crucible/record`

Crucible's `record.Recorder` **is** the galapagos implementation (frame
cap, `done` latch, error on empty save). Only the constructor argument
order changes: `newRecorder(maxFrames, fps, scale)` becomes
`record.NewRecorder(fps, scale, maxFrames)`. Delete the local file.

## 2. `internal/envs/*` renderers

The env visualizers draw through `internal/render/ebiten`; they are
galapagos-specific 2D views, not the raycaster, so they stay. If an env
ever wants pan/zoom, use `crucible/camera` rather than growing a local
one.

## 3. `internal/envs/cube/render.go:80` and friends

Struct `sort.Slice` calls in render paths stay, per the hot-path rule.

## 4. Dependency note

After rubix and gambit migrate, galapagos will pull crucible transitively
through them; its own `go.mod` should still list crucible directly for the
packages it uses first-hand.

## Order of work

1. `record`.
2. `hub`, once the GUI grows a multi-window mode.
