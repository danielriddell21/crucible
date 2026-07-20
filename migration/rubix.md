# rubix → crucible

rubix contributes two of crucible's packages (`record`, `hub`). The cube
model, solvers, EV3 robot driver, and the 3D cube visualizer stay put — as
does `internal/cli`, including `completion.go` (the CLI entrypoint is not
crucible's concern).

## 1. `internal/gui/record.go` → `crucible/record`

Delete the local `recorder` and use `record.Recorder`. The APIs differ
slightly — crucible adopted the galapagos superset:

| Local | Crucible |
|---|---|
| `newRecorder(scale, fps)` | `record.NewRecorder(fps, scale, 0)` — note the argument order, and the third argument is a max-frame cap (0 = unlimited) |
| `r.add(img)` | `r.Add(img)` |
| `r.len()` | `r.Len()` |
| `r.save(path)` | `r.Save(path)` — now fails on an empty recording instead of writing a zero-frame GIF |

`downscale` goes away with the local file.

## 2. `internal/cli/coord.go` + `internal/gui/link.go` → `crucible/hub`

The hub, leader, and child plumbing in `coord.go` (~250 lines) is crucible's
`hub` package, generic over `gui.Msg`. Keep `gui.Msg` where it is; replace
`gui.Link` with `hub.Link[gui.Msg]`.

The `handle` switch becomes a `Route` policy:

```go
cfg := hub.Config[gui.Msg]{
    Self: os.Args[0],
    ChildArgs: func(idx int) []string {
        return []string{"view", fmt.Sprintf("--child=%d", idx)}
    },
    Route: func(m gui.Msg) hub.Route {
        switch m.Type {
        case "state":
            return hub.RouteState
        case "rescramble":
            return hub.RouteBroadcast
        case "add":
            return hub.RouteSpawn
        case "remove":
            return hub.RouteCloseNewest
        }
        return hub.RouteNone
    },
    Quit: gui.Msg{Type: "quit"},
}
```

- `runLeader(ctrl)` becomes `hub.RunLeader(cfg, func(l hub.Link[gui.Msg]) error {
  return gui.Run(gui.Config{Controller: ctrl, Link: &l}) })` (or change
  `gui.Config.Link` to take the value).
- `runChild(ctrl)` becomes `hub.RunChild[gui.Msg](...)` the same way.
- The `_eof` sentinel message type disappears — the hub handles child EOF
  internally, so drop `eofType` and the `drop` case from `Msg` handling.
- `maxWindows = 16` is `hub.DefaultMaxWindows`; delete the local constant.

## 3. Optional

- `pkg/render/render.go:302` and `internal/gui/gui.go:751` use
  `sort.Slice` on structs in draw paths — leave them; crucible deliberately
  keeps hot-path sorts in place.

## Order of work

1. `record` (self-contained, GUI-tag build only).
2. `hub` (touches `internal/cli` wiring and `gui.Config`); run
   `just view` and the multi-window `add`/`remove` keys to verify.
