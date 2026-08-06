package canvas

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/vector"
)

// Polygon fills the closed polygon through pts with col, anti-aliased. Fewer
// than three points draws nothing. Points are in canvas pixel coordinates.
func (c *Canvas) Polygon(pts [][2]float64, col color.RGBA) {
	if len(pts) < 3 {
		return
	}
	c.rasterise(boundsOf(pts), col, func(r *vector.Rasterizer, ox, oy float32) {
		r.MoveTo(float32(pts[0][0])-ox, float32(pts[0][1])-oy)
		for _, p := range pts[1:] {
			r.LineTo(float32(p[0])-ox, float32(p[1])-oy)
		}
		r.ClosePath()
	})
}

// Circle fills a disc of the given radius centred on (x, y), anti-aliased. It
// approximates the outline with enough straight segments that the edge reads
// as smooth at the radius drawn.
func (c *Canvas) Circle(x, y, radius float64, col color.RGBA) {
	if radius <= 0 {
		return
	}
	c.Polygon(circlePoints(x, y, radius), col)
}

// Ring strokes a circle's outline of the given width, anti-aliased — the halo
// round a highlighted entity, a radius marker on a map. A width at or beyond
// the radius fills the disc.
func (c *Canvas) Ring(x, y, radius, width float64, col color.RGBA) {
	if radius <= 0 {
		return
	}
	width = math.Max(width, 1)
	inner := radius - width
	if inner <= 0 {
		c.Circle(x, y, radius, col)
		return
	}
	outer := circlePoints(x, y, radius)
	// The hole is wound the other way, so nonzero filling leaves it empty.
	hole := circlePoints(x, y, inner)
	for i, j := 0, len(hole)-1; i < j; i, j = i+1, j-1 {
		hole[i], hole[j] = hole[j], hole[i]
	}
	c.rasterise(boundsOf(outer), col, func(r *vector.Rasterizer, ox, oy float32) {
		trace(r, outer, ox, oy)
		trace(r, hole, ox, oy)
	})
}

// trace emits a closed contour into the rasteriser, offset to its origin.
func trace(r *vector.Rasterizer, pts [][2]float64, ox, oy float32) {
	r.MoveTo(float32(pts[0][0])-ox, float32(pts[0][1])-oy)
	for _, p := range pts[1:] {
		r.LineTo(float32(p[0])-ox, float32(p[1])-oy)
	}
	r.ClosePath()
}

// Line strokes a line from (x1, y1) to (x2, y2) with the given width,
// anti-aliased. A width below one pixel is drawn as one pixel.
func (c *Canvas) Line(x1, y1, x2, y2, width float64, col color.RGBA) {
	dx, dy := x2-x1, y2-y1
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	// Offset half the stroke width either side of the centre line, giving a
	// quad the rasteriser can fill.
	half := math.Max(width, 1) / 2
	nx, ny := -dy/length*half, dx/length*half
	c.Polygon([][2]float64{
		{x1 + nx, y1 + ny},
		{x2 + nx, y2 + ny},
		{x2 - nx, y2 - ny},
		{x1 - nx, y1 - ny},
	}, col)
}

// circlePoints approximates a circle as a polygon, using roughly one segment
// per two pixels of circumference so the edge stays smooth as radius grows.
func circlePoints(x, y, radius float64) [][2]float64 {
	segments := int(math.Pi * radius)
	segments = min(max(segments, 12), 256)
	pts := make([][2]float64, segments)
	for i := range pts {
		a := 2 * math.Pi * float64(i) / float64(segments)
		pts[i] = [2]float64{x + math.Cos(a)*radius, y + math.Sin(a)*radius}
	}
	return pts
}

// boundsOf returns the pixel rectangle covering pts, rounded outwards so
// anti-aliased edges are not clipped.
func boundsOf(pts [][2]float64) image.Rectangle {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		minX, maxX = math.Min(minX, p[0]), math.Max(maxX, p[0])
		minY, maxY = math.Min(minY, p[1]), math.Max(maxY, p[1])
	}
	return image.Rect(
		int(math.Floor(minX)), int(math.Floor(minY)),
		int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1,
	)
}

// rasterise fills a path into the canvas, anti-aliased and clipped to it. emit
// describes the path in coordinates relative to the clipped bounding box,
// whose top-left it receives, so only the affected pixels are scanned.
func (c *Canvas) rasterise(bounds image.Rectangle, col color.RGBA, emit func(r *vector.Rasterizer, ox, oy float32)) {
	b := bounds.Intersect(c.img.Bounds())
	if b.Empty() {
		return
	}
	r := vector.NewRasterizer(b.Dx(), b.Dy())
	emit(r, float32(b.Min.X), float32(b.Min.Y))
	// Shape colours are given in straight alpha, as [Canvas.Blend] takes them;
	// the rasteriser composites in premultiplied space.
	r.Draw(c.img, b, image.NewUniform(premultiply(col)), image.Point{})
}

// premultiply converts a straight-alpha colour to the premultiplied form Go's
// image compositing expects.
func premultiply(c color.RGBA) color.RGBA {
	if c.A == 0xff {
		return c
	}
	a := uint32(c.A)
	scale := func(v uint8) uint8 { return uint8((uint32(v)*a + 127) / 255) }
	return color.RGBA{R: scale(c.R), G: scale(c.G), B: scale(c.B), A: c.A}
}
