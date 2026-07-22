// Package view is a display-free 2D pan/zoom camera: it maps between world
// and screen space around a centred focus, with follow and fit-to-bounds
// framing, and imports no rendering backend — so headless sims and web
// front-ends share the same camera math the Ebiten front-ends use. When a
// draw transform is needed on top of this, the front-end layers
// [github.com/danielriddell21/crucible/camera] over it.
package view

// Camera frames a 2D world onto a ViewW×ViewH screen, centred on (X, Y) and
// magnified by Zoom.
type Camera struct {
	// X and Y are the world point at the centre of the view.
	X, Y float64
	// Zoom is the magnification factor.
	Zoom float64
	// ViewW and ViewH are the screen size in pixels.
	ViewW, ViewH float64
}

// New returns a camera at unit zoom filling a viewW×viewH screen.
func New(viewW, viewH float64) *Camera {
	return &Camera{Zoom: 1, ViewW: viewW, ViewH: viewH}
}

// WorldToScreen maps a world point to screen coordinates.
func (c *Camera) WorldToScreen(wx, wy float64) (sx, sy float64) {
	return (wx-c.X)*c.Zoom + c.ViewW/2, (wy-c.Y)*c.Zoom + c.ViewH/2
}

// ScreenToWorld maps a screen point back to world coordinates.
func (c *Camera) ScreenToWorld(sx, sy float64) (wx, wy float64) {
	return (sx-c.ViewW/2)/c.Zoom + c.X, (sy-c.ViewH/2)/c.Zoom + c.Y
}

// Follow centres the camera on a world point.
func (c *Camera) Follow(wx, wy float64) { c.X, c.Y = wx, wy }

// FitBounds centres and zooms the camera so the world rectangle, grown by
// margin (a fraction of its size), fits the screen. A degenerate rectangle
// resets the zoom to 1.
func (c *Camera) FitBounds(minX, minY, maxX, maxY, margin float64) {
	c.X = (minX + maxX) / 2
	c.Y = (minY + maxY) / 2
	w := (maxX - minX) * (1 + margin)
	h := (maxY - minY) * (1 + margin)
	if w <= 0 || h <= 0 {
		c.Zoom = 1
		return
	}
	c.Zoom = min(c.ViewW/w, c.ViewH/h)
}
