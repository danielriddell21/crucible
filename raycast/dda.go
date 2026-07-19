package raycast

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
)

// maxSteps bounds the DDA walk so a ray escaping a malformed map still
// terminates.
const maxSteps = 256

// minDist keeps hit distances away from zero so column heights stay finite.
const minDist = 1e-4

// Hit describes where a ray met a solid cell.
type Hit struct {
	// Cell is the solid cell that stopped the ray.
	Cell geom.Coord
	// Prev is the open cell the ray was in just before the hit.
	Prev geom.Coord
	// Dist is the perpendicular distance to the wall face, for column
	// heights and the z-buffer.
	Dist float64
	// Side is 0 for an east/west face and 1 for a north/south face,
	// conventionally shaded darker.
	Side int
	// WallX is the fractional position along the wall face in [0, 1), for
	// texture sampling.
	WallX float64
}

// Cast walks the grid from pos along the ray until solid reports a blocking
// cell, and returns the hit. The walk gives up after crossing 256 cells and
// reports wherever it stopped.
func Cast(pos geom.Vec2, rayX, rayY float64, solid func(x, y int) bool) Hit {
	mapX, mapY := int(math.Floor(pos.X)), int(math.Floor(pos.Y))
	deltaX, deltaY := math.Abs(1/rayX), math.Abs(1/rayY)
	var stepX, stepY int
	var sideX, sideY float64
	if rayX < 0 {
		stepX, sideX = -1, (pos.X-float64(mapX))*deltaX
	} else {
		stepX, sideX = 1, (float64(mapX)+1-pos.X)*deltaX
	}
	if rayY < 0 {
		stepY, sideY = -1, (pos.Y-float64(mapY))*deltaY
	} else {
		stepY, sideY = 1, (float64(mapY)+1-pos.Y)*deltaY
	}
	side := 0
	prevX, prevY := mapX, mapY
	for range maxSteps {
		prevX, prevY = mapX, mapY
		if sideX < sideY {
			sideX += deltaX
			mapX += stepX
			side = 0
		} else {
			sideY += deltaY
			mapY += stepY
			side = 1
		}
		if solid(mapX, mapY) {
			break
		}
	}
	dist := max(BoundaryDist(sideX, sideY, deltaX, deltaY, side), minDist)
	return Hit{
		Cell:  geom.Coord{X: mapX, Y: mapY},
		Prev:  geom.Coord{X: prevX, Y: prevY},
		Dist:  dist,
		Side:  side,
		WallX: BoundaryWallX(pos, rayX, rayY, dist, side),
	}
}

// BoundaryDist converts DDA side distances into the perpendicular distance
// of the boundary just crossed. Games use it directly when they walk the
// grid themselves for custom column logic such as sliding doors.
func BoundaryDist(sideX, sideY, deltaX, deltaY float64, side int) float64 {
	if side == 0 {
		return sideX - deltaX
	}
	return sideY - deltaY
}

// BoundaryWallX returns the fractional position in [0, 1) along the wall
// face where a ray at the given perpendicular distance crosses it.
func BoundaryWallX(pos geom.Vec2, rayX, rayY, dist float64, side int) float64 {
	var wx float64
	if side == 0 {
		wx = pos.Y + dist*rayY
	} else {
		wx = pos.X + dist*rayX
	}
	return wx - math.Floor(wx)
}
