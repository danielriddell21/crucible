package raycast

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
)

// maxWalkSteps bounds the height walk; a ray that never closes its column
// reports an infinite depth.
const maxWalkSteps = 4096

// Heights describes a variable-height world to [WalkColumn]: per-cell
// floor and ceiling heights, solidity, and the optional see-over height of
// half walls. A flat game returns constant 0/1 floors and ceilings.
type Heights interface {
	// Floor returns the walking-surface height of a cell.
	Floor(x, y int) float64
	// Ceil returns the ceiling height of a cell.
	Ceil(x, y int) float64
	// Solid reports whether the cell blocks movement and sight at full
	// height.
	Solid(x, y int) bool
	// WallTop returns the see-over height of a solid cell's half wall, or
	// 0 for a full-height wall.
	WallTop(x, y int) float64
}

// Face describes one vertical surface [WalkColumn] uncovered: a full wall,
// a rising step, or a dropping ceiling edge.
type Face struct {
	// Cell is the cell whose face is visible.
	Cell geom.Coord
	// From is the open cell the ray saw the face from, for theme and
	// light lookups.
	From geom.Coord
	// Dist is the perpendicular distance to the face.
	Dist float64
	// Side is 0 for an east/west face and 1 for a north/south face.
	Side int
	// WallX is the fractional position along the face in [0, 1), for
	// texture sampling.
	WallX float64
	// Flip reports whether the texture column should be mirrored so
	// faces read consistently from both sides.
	Flip bool
}

// ColumnPainter receives the screen spans [WalkColumn] uncovers, top and
// bottom rows inclusive. The painter owns all texturing, lighting, and
// shading; spans arrive already clipped to the column's visible window.
type ColumnPainter interface {
	// WallSpan paints rows y0..y1 of a vertical face.
	WallSpan(x, y0, y1 int, face Face)
	// FloorSpan paints rows y0..y1 of the walking surface of cell, whose
	// height is z.
	FloorSpan(x, y0, y1 int, cell geom.Coord, z, dist float64)
	// CeilSpan paints rows y0..y1 of the ceiling of cell, whose height is
	// z; the game decides whether that cell shows stone or sky.
	CeilSpan(x, y0, y1 int, cell geom.Coord, z, dist float64)
}

// ColumnResult reports what a walked column learned for the rest of the
// frame.
type ColumnResult struct {
	// Depth is the distance the column was closed at, for the sprite
	// z-buffer; +Inf when the ray escaped.
	Depth float64
	// LowZ, LowH, and LowRow describe the nearest see-over ledge the
	// column crossed — its distance, top height, and top screen row — so
	// sprites behind a half wall clip correctly. LowZ is +Inf when the
	// column crossed none.
	LowZ, LowH float64
	LowRow     int
}

// ProjectRow projects world height z at perpendicular distance dist onto a
// screen of h rows viewed from eye height eyeZ.
func ProjectRow(h int, eyeZ, z, dist float64) int {
	return int(float64(h)/2 + (eyeZ-z)*float64(h)/dist)
}

// WalkColumn walks screen column x of a w×h view through a variable-height
// world, calling the painter for every floor, ceiling, and wall span it
// uncovers, nearest first. The camera supplies the ray; eyeZ is the
// viewer's eye height in world units.
//
// The walk keeps a shrinking visible window per column: each cell fills
// the floor and ceiling it shows, each boundary draws its faces, and a
// solid cell either closes the column (full wall) or, for a half wall,
// raises the window floor so the room beyond still draws above it.
func WalkColumn(cam Camera, w, h, x int, eyeZ float64, hg Heights, p ColumnPainter) ColumnResult {
	dx, dy := cam.RayDir(x, w)
	walk := &columnWalk{
		hg: hg, p: p,
		h: h, x: x, eyeZ: eyeZ,
		px: cam.Pos.X, py: cam.Pos.Y, dx: dx, dy: dy,
		yTop: 0, yBot: h - 1,
		res: ColumnResult{LowZ: math.Inf(1)},
	}
	walk.run()
	return walk.res
}

// columnWalk carries one column's state across the DDA steps.
type columnWalk struct {
	hg   Heights
	p    ColumnPainter
	h, x int
	eyeZ float64

	px, py, dx, dy float64

	yTop, yBot    int
	aX, aY        int
	aFloor, aCeil float64
	res           ColumnResult
}

func (c *columnWalk) row(z, d float64) int { return ProjectRow(c.h, c.eyeZ, z, d) }

func (c *columnWalk) run() {
	dda := newDDA(c.px, c.py, c.dx, c.dy)
	c.aX, c.aY = dda.mapX, dda.mapY
	c.aFloor = c.hg.Floor(c.aX, c.aY)
	c.aCeil = c.hg.Ceil(c.aX, c.aY)

	for range maxWalkSteps {
		d, side := dda.advance()
		if c.step(dda.mapX, dda.mapY, d, side) {
			return
		}
	}
	c.res.Depth = math.Inf(1)
}

// step handles one boundary crossing and reports whether the column
// closed.
func (c *columnWalk) step(bX, bY int, d float64, side int) bool {
	floorEdge, ceilEdge := c.paintDeparted(d)
	face := c.boundaryFace(bX, bY, d, side)

	bFloor, bCeil := c.hg.Floor(bX, bY), c.hg.Ceil(bX, bY)
	if c.hg.Solid(bX, bY) {
		top := c.hg.WallTop(bX, bY)
		if top <= 0 {
			c.paintWall(max(c.yTop, ceilEdge+1), min(c.yBot, floorEdge), face)
			c.res.Depth = d
			return true
		}
		// A half wall: a solid block we can see over. Treat it like an
		// unclimbable step up to its top, then keep walking the ray so the
		// room beyond is drawn above it.
		bFloor, bCeil = top, 1
	}

	c.paintFaces(face, floorEdge, ceilEdge, bFloor, bCeil)

	c.yBot = min(c.yBot, c.row(max(c.aFloor, bFloor), d))
	c.yTop = max(c.yTop, c.row(min(c.aCeil, bCeil), d)+1)
	if c.yTop > c.yBot {
		c.res.Depth = d
		return true
	}
	c.aFloor, c.aCeil = bFloor, bCeil
	c.aX, c.aY = bX, bY
	return false
}

// paintDeparted fills the floor and ceiling the departed cell shows up to
// this boundary, returning their screen edges.
func (c *columnWalk) paintDeparted(d float64) (floorEdge, ceilEdge int) {
	from := geom.Coord{X: c.aX, Y: c.aY}
	floorEdge = c.row(c.aFloor, d)
	if y0 := max(c.yTop, floorEdge+1); y0 <= c.yBot {
		c.p.FloorSpan(c.x, y0, c.yBot, from, c.aFloor, d)
	}
	ceilEdge = c.row(c.aCeil, d)
	if y1 := min(c.yBot, ceilEdge); c.yTop <= y1 {
		c.p.CeilSpan(c.x, c.yTop, y1, from, c.aCeil, d)
	}
	return floorEdge, ceilEdge
}

// paintFaces draws the rising step and dropping ceiling faces of an open
// (or see-over) boundary and tracks the nearest see-over ledge.
func (c *columnWalk) paintFaces(face Face, floorEdge, ceilEdge int, bFloor, bCeil float64) {
	if bFloor > c.aFloor { // rising step face — a half wall, stair, lift or ledge
		stepRow := c.row(bFloor, face.Dist)
		c.paintWall(max(c.yTop, stepRow+1), min(c.yBot, floorEdge), face)
		if bFloor > c.res.LowH {
			c.res.LowH, c.res.LowZ, c.res.LowRow = bFloor, face.Dist, stepRow
		}
	}
	if bCeil < c.aCeil { // dropping ceiling face
		c.paintWall(max(c.yTop, ceilEdge+1), min(c.yBot, c.row(bCeil, face.Dist)), face)
	}
}

func (c *columnWalk) paintWall(y0, y1 int, face Face) {
	if y0 <= y1 {
		c.p.WallSpan(c.x, y0, y1, face)
	}
}

func (c *columnWalk) boundaryFace(bX, bY int, d float64, side int) Face {
	var wx float64
	if side == 0 {
		wx = c.py + d*c.dy
	} else {
		wx = c.px + d*c.dx
	}
	return Face{
		Cell:  geom.Coord{X: bX, Y: bY},
		From:  geom.Coord{X: c.aX, Y: c.aY},
		Dist:  d,
		Side:  side,
		WallX: wx - math.Floor(wx),
		Flip:  (side == 0 && c.dx > 0) || (side == 1 && c.dy < 0),
	}
}

// dda walks tile boundaries along a ray.
type dda struct {
	mapX, mapY           int
	stepX, stepY         int
	sideDistX, sideDistY float64
	deltaX, deltaY       float64
}

func newDDA(px, py, dx, dy float64) *dda {
	d := &dda{
		mapX:   int(math.Floor(px)),
		mapY:   int(math.Floor(py)),
		deltaX: math.Inf(1),
		deltaY: math.Inf(1),
	}
	if dx != 0 {
		d.deltaX = math.Abs(1 / dx)
	}
	if dy != 0 {
		d.deltaY = math.Abs(1 / dy)
	}
	if dx < 0 {
		d.stepX, d.sideDistX = -1, (px-float64(d.mapX))*d.deltaX
	} else {
		d.stepX, d.sideDistX = 1, (float64(d.mapX)+1-px)*d.deltaX
	}
	if dy < 0 {
		d.stepY, d.sideDistY = -1, (py-float64(d.mapY))*d.deltaY
	} else {
		d.stepY, d.sideDistY = 1, (float64(d.mapY)+1-py)*d.deltaY
	}
	return d
}

// advance crosses the next tile boundary and returns its perpendicular
// distance and side.
func (d *dda) advance() (dist float64, side int) {
	if d.sideDistX < d.sideDistY {
		dist = d.sideDistX
		d.sideDistX += d.deltaX
		d.mapX += d.stepX
	} else {
		dist, side = d.sideDistY, 1
		d.sideDistY += d.deltaY
		d.mapY += d.stepY
	}
	return max(dist, 1e-6), side
}
