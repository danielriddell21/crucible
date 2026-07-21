# gambit → crucible

gambit's chess engine, agents, board GUI, and `internal/cli` stay put; the
migration is the recorder and one honest ordinex/retrievium call site.

## 1. `internal/gui/record.go` → `crucible/record`

Same swap as rubix (see rubix.md §1): delete the local recorder, use
`record.NewRecorder(fps, scale, maxFrames)` / `Add` / `Len` / `Save`.
gambit's copy already matched the galapagos-style signature crucible
adopted, so only the constructor argument order changes
(`newRecorder(maxFrames, fps, scale)` → `NewRecorder(fps, scale, maxFrames)`).

## 2. `internal/agent/agent.go:48` — ordinex/retrievium fit

The agent registry sorts its names with `sort.Strings` and looks names up
linearly. Both are cold paths and plain ordered slices — the exact fit for
the algorithm libraries (both stay gambit dependencies; neither is pulled
in through crucible):

```go
names = ordinex.MergeSorter[string]{}.Sort(names)          // returns a copy
if _, ok := (retrievium.BinarySearcher[string]{}).Search(names, want); ok { ... }
```

This is retrievium's honest home. Its searchers do *exact-match* membership
("is `want` in this sorted slice, and where"), which is precisely a
name→agent lookup. That's a genuine but app-local fit, so retrievium stays
a gambit dependency and is deliberately **not** a crucible dependency: the
engine's searchable lists (tile kinds, window ids, a few shells) are all
small enough that a linear scan is as good, so a binary-search dependency
in the engine would be decoration, not value. The one search the engine
genuinely wants is threshold selection — "pick by cumulative weight" — which
is a *different* search (upper-bound, not exact-match) and lives in crucible
as `worldgen.WeightedChoice` on stdlib `sort.Search`.

## 3. Stays put

- `internal/gui/game_ui.go`, `render.go`, `glyphs.go` — board drawing is
  gambit-specific (crucible's `canvas` is for the raycaster-style text
  screens, which gambit doesn't use).
- `internal/agent/minimax.go:96`, `beam.go:87` — `sort.SliceStable` over
  scored moves in the search hot path; keep.
- `internal/log` — three dozen lines, not worth a dependency.

## Order of work

1. `record`.
2. Agent registry ordinex/retrievium swap.
