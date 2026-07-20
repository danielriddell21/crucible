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

ordinex and retrievium have no guide: nothing migrates out of them — crucible
consumes them as dependencies (hub window ids), and the
app guides call out further call sites where their interfaces fit.

## Ground rules (all repos)

- Add the dependency: `go get github.com/danielriddell21/crucible@latest`.
- Migrate one package at a time; `just ci` must stay green after each step.
- Behaviour must not change: these are extractions, not redesigns. Where a
  crucible API differs from the local copy (noted per guide), adapt the call
  site, not the behaviour.
- Keep each repo's `internal/cli` command tree and `internal/gui` seam
  (`Run(Config)`/`Available()`, `//go:build ebiten` stubs) exactly as
  CONVENTIONS.md describes — crucible slots in behind them.
- README, CI, lint rules, and justfiles stay as they are; no repo gains new
  workflows from this migration. The only expected diff outside `.go` files
  is `go.mod`/`go.sum`.
- Until crucible's tag workflow has cut a release covering what you need,
  pin the trunk pseudo-version the same way crucible pins ordinex and
  retrievium.
