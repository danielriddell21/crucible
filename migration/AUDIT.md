# Consolidation audit

A second pass over the seven family repos after the first round of guides,
answering one question for every remaining piece of shared-looking code:
**can it move into crucible, and if not, exactly why?** The rule of thumb,
in order:

1. If two or more repos carry the same display-free logic, extract it.
2. If it is nearly the same but carries app-defined values, make the engine
   side generic and let each app keep its vocabulary.
3. If it is not worth genericising, say why, and note which existing crucible
   package it sits behind.

## Extracted this pass

| Package | What moved | Repos | Notes |
|---|---|---|---|
| `rng` | `Stream(seed, stream)` — one PCG sub-stream per subsystem | galapagos, hegemony | Both had a byte-identical `newStream`/stream-id helper. The stream **ids** stay app-side (they are each game's vocabulary); only the constructor moved. |
| `view` | Display-free 2D pan/zoom camera (`WorldToScreen`, `Follow`, `FitBounds`) | galapagos | Byte-identical to galapagos `core.Camera`; adopted as a type alias so call sites are untouched. `camera` now layers the Ebiten draw transform over this. |
| `ring` | `Ring[T]` fixed-capacity rolling history | galapagos, vivarium | galapagos had a generic `Ring[T]`; vivarium hand-rolled the same bounded-slice idiom twice (`history`, `lineageHistory`). One generic type replaces all three. |
| `paint` | `Scale`/`ScaleBytes` colour brightness, `BlendOver` full-frame tint | pandemonium, nemesis | The one pixel kernel both software raycasters shared verbatim (`scaleColor`/`shadeRGBA`/`shade` tails, `dimmed`/`shadeBytes`, `blendOver`/`tint`). Golden render tests confirm byte-identity. |
| `raycast.Camera.ProjectAt` | Height-aware billboard projection | pandemonium | Sprite screen placement that used to be inlined; now a tested method beside `Project`. nemesis keeps its custom sliding-door ray loop but shares the camera. |

Distance-shading **curves** were deliberately left behind with `paint`: how
brightness falls off with distance is art direction, and the two games use
different formulas (`1/(1+d·k)` vs `light·1.25/(1+d²·k)`) with different
constants. Each game keeps its `shadeFactor` and calls `paint.Scale` with the
result.

## Stays app-side — and why

### HUD / panel rendering (galapagos, vivarium, hegemony, gambit, rubix)
Every front-end draws its own panels (info panels, keymaps, inspectors, stat
bars, sparklines). Two reasons they do not consolidate into a drawing package:

- **The Ebiten rule.** Only `menu` and `camera` may import Ebiten. A panel
  renderer that strokes rects and text needs a backend, so it cannot become a
  third display package without breaking the constraint the whole engine is
  built around.
- **They are genuinely bespoke.** The data differs per app (agent vitals,
  faction tallies, cube state), and each already routes through its own seam —
  galapagos through a `Renderer` interface, vivarium through `drawPanel`/
  `drawText` helpers. There is no shared *logic*, only a shared *look*.

The one display-free kernel hiding behind the panels — the rolling history a
sparkline plots — **was** extracted (`ring`). For text-screen composition
that does not sit on a live GL frame (menus, end cards), `canvas` already
provides display-free `Rect`/`Text`/`TextCentered`.

### Sprite frame / direction picking (pandemonium, nemesis)
Both games choose a sprite frame per entity, but the schemes are different
vocabularies: pandemonium indexes a `bands × dirs` face table; nemesis picks
an `alienFrame` from game state and time. No shared kernel — this is gameplay
presentation, which the engine rules keep app-side.

### Viewmodel / muzzle flash (pandemonium)
Only one game has a first-person weapon. Nothing to consolidate against.

### Notice / narration text (pandemonium, nemesis)
`notice()` and `Event.Line()` turn game events into player-facing strings.
This is exactly the app-defined message vocabulary the engine rules forbid
crucible from growing. The **mechanism** is already shared: both games wrap
`telemetry.Bus[Event]` and post lines through `hud.Overlay` /
`status` / `narrate`. Only the words stay home.

### Telemetry event types (pandemonium, nemesis, galapagos)
Each `Event`/`Observation` type is app vocabulary. The generic fan-out —
subscriber registration, publish filter, bounded recent feed — is already
`telemetry.Bus`, which nemesis and pandemonium wrap. galapagos's `Telemetry`
is a thin generation-stats logger over `ring`; its `GenStats` shape is
app-specific and stays put.

### Neural networks (vivarium `neural`, galapagos `nn`)
The two evolve genuinely different nets (recurrent per-agent brains with a
forward model vs. feed-forward NEAT genomes). The shared surface is thin and
the shapes diverge; forcing a common package would be an abstraction the rule
of "a focused function over a clever abstraction" warns against. Left
separate.

## Nothing further from the pure-consumer repos
ordinex (sorted window ids) and retrievium (sorted-slice search) remain
crucible's/apps' dependencies rather than migration sources, unchanged from
the first-round guides.
