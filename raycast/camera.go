// Package raycast is the software raycasting core behind the family's
// first-person games: a planar camera, the DDA grid walk that finds the
// wall each screen column hits, and the billboard projection used to place
// sprites against the wall z-buffer.
//
// A [Camera] (from [NewCamera]) aims a ray per screen column with
// [Camera.RayDir]; [Cast] walks the grid and returns a [Hit] carrying the
// perpendicular distance, face side, and texture coordinate. Sprites
// project through [Camera.Project] into a [Placement] and draw far-to-near
// after [SortFarToNear].
//
// The package computes geometry only. Texturing, shading, and the
// framebuffer belong to each game's renderer, and game-specific column
// logic (sliding doors, variable heights) builds on the exported
// [BoundaryDist] and [BoundaryWallX] helpers.
package raycast

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
)

// Camera is a raycasting camera: a position, a facing direction, and the
// screen plane perpendicular to it.
type Camera struct {
	Pos            geom.Vec2
	DirX, DirY     float64
	PlaneX, PlaneY float64
}

// NewCamera returns a camera at pos facing angle radians, with the given
// horizontal field of view in radians.
func NewCamera(pos geom.Vec2, angle, fov float64) Camera {
	dirX, dirY := math.Cos(angle), math.Sin(angle)
	planeLen := math.Tan(fov / 2)
	return Camera{
		Pos:    pos,
		DirX:   dirX,
		DirY:   dirY,
		PlaneX: -dirY * planeLen,
		PlaneY: dirX * planeLen,
	}
}

// RayDir returns the direction of the ray through screen column x of width
// columns.
func (c Camera) RayDir(x, width int) (float64, float64) {
	cameraX := 2*float64(x)/float64(width) - 1
	return c.DirX + c.PlaneX*cameraX, c.DirY + c.PlaneY*cameraX
}
