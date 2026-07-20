# hegemony → crucible

hegemony's territory sim, strategies, tournament, and evolution stay put,
as does `internal/cli` (entrypoint and completion). The migration is the
multi-window hub — its `internal/cli/coord.go` is the rubix design
re-typed.

## 1. `internal/cli/coord.go` + `internal/gui/link.go` → `crucible/hub`

Same swap as rubix (see rubix.md §2), with hegemony's own `gui.Msg` as the
type parameter and its own `Route` policy mapping its message types onto
`hub.RouteState` / `RouteBroadcast` / `RouteSpawn` / `RouteCloseNewest`.
`gui.Msg` and `StateMsg` stay in `internal/gui`; `gui.Link` becomes
`hub.Link[gui.Msg]`.

The child-spawn arguments come from the current `spawnChild` call —
whatever subcommand and `--child=N` flag `coord.go` passes today goes into
`Config.ChildArgs` verbatim.

## 2. `internal/gui/gui_ebiten.go:252`

`sort.SliceStable` in the draw path stays, per the hot-path rule.

## 3. Stays put

- The `gui_stub.go` / `//go:build ebiten` seam — unchanged.
- `internal/sim/rng.go` — hegemony's sim RNG is not the worldgen RNG;
  leave it unless the sim ever adopts `worldgen`.

## Order of work

1. `hub`; verify with a leader window spawning and closing children.
