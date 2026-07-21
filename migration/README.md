# Migration guides

One guide per family repo, mapping its current code to crucible packages.
No repo has been changed; each guide is the worklist for doing so.

| Guide | Repo | Reviewed at |
|---|---|---|
| [rubix.md](rubix.md) | github.com/danielriddell21/rubix | trunk |
| [gambit.md](gambit.md) | github.com/danielriddell21/gambit | trunk |
| [vivarium.md](vivarium.md) | github.com/danielriddell21/vivarium | trunk |
| [galapagos.md](galapagos.md) | github.com/danielriddell21/galapagos | trunk |
| [hegemony.md](hegemony.md) | github.com/danielriddell21/hegemony | trunk |
| [pandemonium.md](pandemonium.md) | github.com/danielriddell21/pandemonium | **feat/audio-cues** |
| [nemesis.md](nemesis.md) | github.com/danielriddell21/nemesis | trunk |

ordinex has no guide: nothing migrates out of it — crucible consumes it as a
dependency (the `hub` sorts window ids with it), and the app guides call out
further call sites where its interface fits. retrievium is not a crucible
dependency; the guides note where its sorted-slice search fits each app.

## Ground rules (all repos)

- Work on a `refactor/adopt-crucible` branch off trunk, one crucible package
  at a time; `just ci` (both build tags) must stay green after each step, and
  smoke-run the real app before pushing.
- Add the dependency: `go get github.com/danielriddell21/crucible@v0.1.0`.
- Behaviour must not change: these are extractions, not redesigns. Where a
  crucible API differs from the local copy (noted per guide), adapt the call
  site, not the behaviour. One deliberate exception: nemesis's vent networks
  were improved during extraction (nemesis.md §8), so vent layouts differ
  for a given seed.
- Keep each repo's `internal/cli` command tree and `internal/gui` seam
  (`Run(Config)`/`Available()`, `//go:build ebiten` stubs) exactly as
  CONVENTIONS.md describes — crucible slots in behind them.
- README, CI, lint rules, and justfiles stay as they are; no repo gains new
  workflows from this migration. The only expected diff outside `.go` files
  is `go.mod`/`go.sum`.
- **wrapcheck.** A call that used to return a local method's error now
  returns one from an external package (crucible), so the `wrapcheck` linter
  will flag it. Wrap it: `fmt.Errorf("…: %w", err)`. This bites the recorder
  save (`record.Save`) and the hub runners (`hub.RunLeader`/`RunChild`), and
  anywhere else a crucible error is returned unwrapped. Errors that are only
  logged, not returned, are unaffected.
- Pin the tagged release: `go get github.com/danielriddell21/crucible@v0.1.0`
  (crucible in turn pins the tagged `ordinex/v2`, so the graph stays clean).
