# crucible — Claude Code instructions

crucible is a Go library: the shared engine behind the family's Ebitengine apps and games. One package per engine concern at the module root; generic over each app's own types. Correctness and simplicity over cleverness — prefer a well-tested, focused function over a clever abstraction. If the standard library does it, use it.

## Before every commit

* Run `just ci` autonomously (lint + test + build). All must pass.
* When adding a function or package, write unit tests alongside the code in the same commit.
* The public API keeps full godoc coverage — the revive `exported` rule enforces it. Document every exported symbol you add.
* Never commit until the user explicitly confirms. Propose changes as diffs, run `just ci` autonomously, then stop and wait before `git commit`.

## Code quality

* Run `just lint` before proposing a diff. Fix all lint errors before committing.
* Prefer early returns over nesting.
* Do not add error handling or fallbacks for scenarios that cannot happen.
* Public library code keeps full godoc — a doc comment on every exported symbol and the package, enforced by the revive `exported` rule.

## Engine rules

* Only `menu`, `camera`, and `window` may import Ebiten; everything else stays display-free.
* Packages that carry app-defined values stay generic — never grow an engine-side message, event, or cue vocabulary. The one engine-owned vocabulary is `level`'s spatial one (tiles, heights, lifts, vents); gameplay meaning (items, markers, hazards, door state) stays in each game.
* Anything random takes an explicit seed.
* Tests that import Ebiten need a display: run `xvfb-run -a just test` on a headless machine.
* `docs/provenance.md` records where each package came from and what deliberately stayed app-side; keep it current when the package set changes.
