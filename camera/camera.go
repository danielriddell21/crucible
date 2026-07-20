// Package camera provides the 2D pan-and-zoom camera the family's top-down
// visualizers use: zoom anchored under the cursor, panning in world units,
// and clamping so the view never leaves the world.
//
// Create a [Camera] with [New], apply [Camera.ZoomAt] and [Camera.Pan]
// from input, and draw through [Camera.GeoM]; [Camera.ScreenToWorld] maps
// cursor positions back into the world. This package imports Ebiten (for
// the GeoM), so in the family's apps it belongs behind the internal/gui
// seam.
package camera

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/danielriddell21/crucible/geom"
)

// Camera is a 2D view over a world of fixed size. The zero value is not
// ready to use; construct one with [New].
type Camera struct {
	// X, Y is the world position of the view's top-left corner.
	X, Y float64
	// Zoom is the magnification factor.
	Zoom float64
	// MinZoom and MaxZoom bound Zoom; [New] sets the conventional 1..12.
	MinZoom, MaxZoom float64
}

// New returns a camera at the origin with no zoom and the conventional
// zoom bounds.
func New() Camera { return Camera{Zoom: 1, MinZoom: 1, MaxZoom: 12} }

// GeoM returns the world-to-screen transform for ebiten draw options.
func (c Camera) GeoM() ebiten.GeoM {
	var g ebiten.GeoM
	g.Translate(-c.X, -c.Y)
	g.Scale(c.Zoom, c.Zoom)
	return g
}

// ScreenToWorld converts a screen position to world coordinates.
func (c Camera) ScreenToWorld(sx, sy float64) (float64, float64) {
	return sx/c.Zoom + c.X, sy/c.Zoom + c.Y
}

// ZoomAt multiplies the zoom by factor while keeping the world point under
// the screen position (sx, sy) fixed, then clamps the view to the world.
func (c *Camera) ZoomAt(factor, sx, sy, worldW, worldH float64) {
	wx, wy := c.ScreenToWorld(sx, sy)
	c.Zoom = geom.Clamp(c.Zoom*factor, c.MinZoom, c.MaxZoom)
	// Re-anchor so the same world point stays under the cursor.
	c.X = wx - sx/c.Zoom
	c.Y = wy - sy/c.Zoom
	c.clamp(worldW, worldH)
}

// Pan moves the view by a screen-space delta, then clamps it to the world.
func (c *Camera) Pan(dx, dy, worldW, worldH float64) {
	c.X += dx / c.Zoom
	c.Y += dy / c.Zoom
	c.clamp(worldW, worldH)
}

func (c *Camera) clamp(worldW, worldH float64) {
	maxX := worldW - worldW/c.Zoom
	maxY := worldH - worldH/c.Zoom
	c.X = geom.Clamp(c.X, 0, maxX)
	c.Y = geom.Clamp(c.Y, 0, maxY)
}
