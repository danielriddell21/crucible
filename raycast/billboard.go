package raycast

import "github.com/danielriddell21/crucible/geom"

// nearPlane rejects billboards closer than this camera-space depth.
const nearPlane = 0.1

// minSize rejects billboards that would project smaller than this many
// pixels.
const minSize = 2

// Placement is a billboard projected into screen space.
type Placement struct {
	// ScreenX is the column of the billboard's centre.
	ScreenX int
	// Top is the y of the billboard's top edge; it may be negative.
	Top int
	// Size is the billboard's height in pixels.
	Size int
	// Depth is the camera-space depth, comparable against a wall z-buffer.
	Depth float64
}

// Project places a world position onto a w×h screen as a billboard scaled
// by scale, reporting false when it is behind the camera or too small to
// draw.
func (c Camera) Project(p geom.Vec2, w, h int, scale float64) (Placement, bool) {
	relX := p.X - c.Pos.X
	relY := p.Y - c.Pos.Y

	// Inverse camera transform into screen space.
	invDet := 1 / (c.PlaneX*c.DirY - c.DirX*c.PlaneY)
	transX := invDet * (c.DirY*relX - c.DirX*relY)
	transY := invDet * (-c.PlaneY*relX + c.PlaneX*relY)
	if transY <= nearPlane {
		return Placement{}, false
	}
	screenX := int(float64(w) / 2 * (1 + transX/transY))
	size := int(float64(h) / transY * scale)
	if size < minSize {
		return Placement{}, false
	}
	top := h/2 + int(float64(h)/transY)/2 - size
	return Placement{ScreenX: screenX, Top: top, Size: size, Depth: transY}, true
}

// SortFarToNear orders items by descending depth so nearer billboards
// overdraw farther ones. It sorts in place with a simple insertion sort:
// billboard counts are small and the call sits on the per-frame hot path,
// so no allocation is worth it here.
func SortFarToNear[T any](items []T, depth func(T) float64) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && depth(items[j-1]) < depth(items[j]); j-- {
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}
