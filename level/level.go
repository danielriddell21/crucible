package level

import (
	"strings"

	"github.com/danielriddell21/crucible/geom"
)

// Level is a tile-grid world with per-cell heights and lighting. W and H
// are the grid size in cells; the per-cell slices are indexed by
// [Level.Index]. Construct with [New] — the zero value has no storage.
type Level struct {
	// W and H are the grid dimensions in cells.
	W, H int
	// Tiles is the tile per cell.
	Tiles []Tile
	// FloorH and CeilH are the walking surface and ceiling heights per
	// cell, in world units where a full wall is 1 tall.
	FloorH, CeilH []float64
	// WallTopH, when positive on a wall cell, lowers that wall to a
	// see-over half wall of the given height.
	WallTopH []float64
	// Light is the per-cell light level in [0, 1].
	Light []float64
	// Sky marks cells open to the sky instead of a ceiling.
	Sky []bool
	// Theme is the per-cell room theme index, assigned by [AssignThemes].
	Theme []uint8
	// Spawn and Exit are the player's start and the level goal.
	Spawn, Exit geom.Coord
	// Lifts maps a cell to its moving platform, keyed by the cell the
	// platform serves.
	Lifts map[geom.Coord]Lift
	// VentMouths lists the vent cells that open onto walkable ground,
	// maintained by [CarveVents].
	VentMouths []geom.Coord
	// Seed is the seed the level was generated from.
	Seed int64
}

// New returns a level of solid wall: every cell is [TileWall] with floor 0,
// ceiling 1, and full light, ready for [worldgen.Generate] to dig into.
func New(w, h int, seed int64) *Level {
	tiles := make([]Tile, w*h)
	ceils := make([]float64, w*h)
	light := make([]float64, w*h)
	for i := range tiles {
		tiles[i] = TileWall
		ceils[i] = 1
		light[i] = 1
	}
	return &Level{
		W:        w,
		H:        h,
		Tiles:    tiles,
		FloorH:   make([]float64, w*h),
		CeilH:    ceils,
		WallTopH: make([]float64, w*h),
		Light:    light,
		Sky:      make([]bool, w*h),
		Theme:    make([]uint8, w*h),
		Seed:     seed,
	}
}

// Index returns the slice index of cell (x, y). The cell must be in
// bounds.
func (l *Level) Index(x, y int) int { return y*l.W + x }

// InBounds reports whether (x, y) lies on the grid.
func (l *Level) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < l.W && y < l.H
}

// At returns the tile at (x, y); out-of-bounds cells read as [TileWall].
func (l *Level) At(x, y int) Tile {
	if !l.InBounds(x, y) {
		return TileWall
	}
	return l.Tiles[l.Index(x, y)]
}

// Set writes the tile at (x, y); out-of-bounds writes are ignored.
func (l *Level) Set(x, y int, t Tile) {
	if l.InBounds(x, y) {
		l.Tiles[l.Index(x, y)] = t
	}
}

// Solid reports whether the cell blocks movement.
func (l *Level) Solid(x, y int) bool { return !l.At(x, y).Walkable() }

// Floor returns the walking-surface height at (x, y); out of bounds is 0.
func (l *Level) Floor(x, y int) float64 {
	if !l.InBounds(x, y) {
		return 0
	}
	return l.FloorH[l.Index(x, y)]
}

// SetFloor writes the walking-surface height at (x, y).
func (l *Level) SetFloor(x, y int, z float64) {
	if l.InBounds(x, y) {
		l.FloorH[l.Index(x, y)] = z
	}
}

// Ceil returns the ceiling height at (x, y); out of bounds is 1.
func (l *Level) Ceil(x, y int) float64 {
	if !l.InBounds(x, y) {
		return 1
	}
	return l.CeilH[l.Index(x, y)]
}

// SetCeil writes the ceiling height at (x, y).
func (l *Level) SetCeil(x, y int, z float64) {
	if l.InBounds(x, y) {
		l.CeilH[l.Index(x, y)] = z
	}
}

// WallTop returns the half-wall height at (x, y), or 0 for a full-height
// wall. With [Level.Floor], [Level.Ceil], and [Level.Solid] it satisfies
// the raycast package's Heights interface, so a level plugs straight into
// its height-aware column walker.
func (l *Level) WallTop(x, y int) float64 {
	if !l.InBounds(x, y) {
		return 0
	}
	return l.WallTopH[l.Index(x, y)]
}

// LightAt returns the light level at (x, y); out of bounds is fully lit.
func (l *Level) LightAt(x, y int) float64 {
	if !l.InBounds(x, y) {
		return 1
	}
	return l.Light[l.Index(x, y)]
}

// SkyAt reports whether the cell is open to the sky.
func (l *Level) SkyAt(x, y int) bool {
	return l.InBounds(x, y) && l.Sky[l.Index(x, y)]
}

// ThemeAt returns the room theme index at (x, y); out of bounds is theme
// 0.
func (l *Level) ThemeAt(x, y int) uint8 {
	if !l.InBounds(x, y) {
		return 0
	}
	return l.Theme[l.Index(x, y)]
}

// Width returns the grid width; with [Level.Height], [Level.Open], and
// [Level.Carve] it satisfies [worldgen.Carver].
func (l *Level) Width() int { return l.W }

// Height returns the grid height.
func (l *Level) Height() int { return l.H }

// Open reports whether the cell has been dug walkable.
func (l *Level) Open(x, y int) bool { return l.At(x, y).Walkable() }

// Carve digs the cell into open floor.
func (l *Level) Carve(x, y int) { l.Set(x, y, TileFloor) }

// String renders the tile grid as one rune per cell, rows separated by
// newlines — the conventional map dump.
func (l *Level) String() string {
	var b strings.Builder
	b.Grow((l.W + 1) * l.H)
	for y := range l.H {
		for x := range l.W {
			b.WriteRune(l.At(x, y).Rune())
		}
		if y < l.H-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
