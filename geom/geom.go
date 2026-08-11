// Package geom provides the small geometric vocabulary shared by the
// Ebitengine tool family: float vectors for simulation space, integer
// coordinates and rectangles for tile grids, and clamping helpers.
//
// [Vec2] is the continuous-space vector, with the usual arithmetic plus the
// toroidal-world helpers ([Vec2.WrapTo], [Vec2.ShortestDelta],
// [Vec2.ToroidalDist]) the ecosystem sims rely on. [Vec3] is its
// three-dimensional counterpart for the Y-up worlds, where the ground is the
// XZ plane and [Vec3.Ground] drops back to a [Vec2]. [Coord] addresses a tile
// grid cell and [Rect] an axis-aligned span of cells; [Coord.Vec2] bridges
// the two spaces at cell centres. [Clamp] and [Clamp01] are the range
// limiters repeated across every settings screen and camera in the family,
// and [WrapPi] and [AngleDiff] the angle folding every steering and turning
// controller in it needs.
package geom

import (
	"cmp"
	"math"
)

// Vec2 is a two-dimensional vector in continuous (simulation or screen)
// space.
type Vec2 struct {
	X, Y float64
}

// Add returns v + o.
func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }

// Sub returns v - o.
func (v Vec2) Sub(o Vec2) Vec2 { return Vec2{v.X - o.X, v.Y - o.Y} }

// Scale returns v with both components multiplied by s.
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }

// Dot returns the dot product of v and o.
func (v Vec2) Dot(o Vec2) float64 { return v.X*o.X + v.Y*o.Y }

// Len returns the Euclidean length of v.
func (v Vec2) Len() float64 { return math.Hypot(v.X, v.Y) }

// Dist returns the Euclidean distance between v and o.
func (v Vec2) Dist(o Vec2) float64 { return v.Sub(o).Len() }

// DistSq returns the squared distance between v and o. It avoids the square
// root, so prefer it for comparisons.
func (v Vec2) DistSq(o Vec2) float64 {
	d := v.Sub(o)
	return d.Dot(d)
}

// Normalize returns v scaled to unit length. The zero vector is returned
// unchanged.
func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l == 0 {
		return v
	}
	return Vec2{v.X / l, v.Y / l}
}

// Angle returns the angle of v in radians, in (-π, π].
func (v Vec2) Angle() float64 { return math.Atan2(v.Y, v.X) }

// FromAngle returns the unit vector pointing at angle a radians.
func FromAngle(a float64) Vec2 { return Vec2{math.Cos(a), math.Sin(a)} }

// WrapTo wraps v into the rectangle [0, w) × [0, h), for toroidal worlds.
func (v Vec2) WrapTo(w, h float64) Vec2 {
	x := math.Mod(v.X, w)
	if x < 0 {
		x += w
	}
	y := math.Mod(v.Y, h)
	if y < 0 {
		y += h
	}
	return Vec2{x, y}
}

// ShortestDelta returns the shortest vector from v to o on a torus of the
// given size.
func (v Vec2) ShortestDelta(o Vec2, w, h float64) Vec2 {
	dx := wrapDelta(o.X-v.X, w)
	dy := wrapDelta(o.Y-v.Y, h)
	return Vec2{dx, dy}
}

// ToroidalDist returns the shortest distance between v and o on a torus of
// the given size.
func (v Vec2) ToroidalDist(o Vec2, w, h float64) float64 {
	return v.ShortestDelta(o, w, h).Len()
}

func wrapDelta(d, size float64) float64 {
	d = math.Mod(d, size)
	if d < -size/2 {
		d += size
	} else if d >= size/2 {
		d -= size
	}
	return d
}

// Vec3 is a three-dimensional vector. The family's 3D worlds are Y-up, with
// the ground on the XZ plane, so [Vec3.Ground] drops the height to give the
// [Vec2] the 2D helpers work on.
type Vec3 struct {
	X, Y, Z float64
}

// Add returns v + o.
func (v Vec3) Add(o Vec3) Vec3 { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }

// Sub returns v - o.
func (v Vec3) Sub(o Vec3) Vec3 { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }

// Scale returns v with every component multiplied by s.
func (v Vec3) Scale(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }

// Dot returns the dot product of v and o.
func (v Vec3) Dot(o Vec3) float64 { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

// Cross returns the cross product of v and o.
func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

// Len returns the length of v.
func (v Vec3) Len() float64 { return math.Sqrt(v.Dot(v)) }

// Dist returns the distance between v and o.
func (v Vec3) Dist(o Vec3) float64 { return v.Sub(o).Len() }

// Normalize returns v scaled to unit length. The zero vector is returned
// unchanged.
func (v Vec3) Normalize() Vec3 {
	l := v.Len()
	if l == 0 {
		return v
	}
	return Vec3{v.X / l, v.Y / l, v.Z / l}
}

// Ground returns the horizontal components of v, dropping its height.
func (v Vec3) Ground() Vec2 { return Vec2{v.X, v.Z} }

// Coord is an integer cell position on a tile grid.
type Coord struct {
	X, Y int
}

// Add returns c translated by o.
func (c Coord) Add(o Coord) Coord { return Coord{c.X + o.X, c.Y + o.Y} }

// Vec2 returns the centre of the cell in continuous space.
func (c Coord) Vec2() Vec2 { return Vec2{float64(c.X) + 0.5, float64(c.Y) + 0.5} }

// Rect is an axis-aligned integer rectangle on a tile grid, described by its
// top-left corner and size.
type Rect struct {
	X, Y, W, H int
}

// Center returns the cell at the centre of the rectangle.
func (r Rect) Center() Coord { return Coord{r.X + r.W/2, r.Y + r.H/2} }

// Contains reports whether the cell c lies inside the rectangle.
func (r Rect) Contains(c Coord) bool {
	return c.X >= r.X && c.X < r.X+r.W && c.Y >= r.Y && c.Y < r.Y+r.H
}

// Clamp returns v limited to the range [lo, hi].
func Clamp[T cmp.Ordered](v, lo, hi T) T {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Clamp01 returns v limited to the range [0, 1].
func Clamp01(v float64) float64 { return Clamp(v, 0, 1) }

// Float is the floating-point width the angle helpers accept. They are
// generic over it because the family's sims are not all float64: a driving
// sim carrying float32 positions would otherwise cast through float64 twice
// on every steering correction.
type Float interface {
	~float32 | ~float64
}

// WrapPi folds an angle in radians into the range [-π, π].
func WrapPi[F Float](a F) F {
	const twoPi = 2 * math.Pi
	w := math.Mod(float64(a), twoPi)
	if w > math.Pi {
		w -= twoPi
	}
	if w < -math.Pi {
		w += twoPi
	}
	return F(w)
}

// AngleDiff returns the signed shortest rotation from b to a, in [-π, π]. It
// is positive when a lies anticlockwise of b, so a steering controller can
// use it directly as an error term.
func AngleDiff[F Float](a, b F) F { return WrapPi(a - b) }

// MoveToward steps current toward target by at most step, stopping exactly on
// target rather than overshooting it. Use it when the rate of change is what
// matters — a dial that may move so far per second, a value that must not jump.
func MoveToward[F Float](current, target, step F) F {
	d := target - current
	if d <= step && -d <= step {
		return target
	}
	if d > 0 {
		return current + step
	}
	return current - step
}

// Approach eases current toward target at the given rate, covering the same
// fraction of the remaining gap per second however long dt is. Use it when the
// shape of the motion is what matters — a camera settling behind a car, a
// remote player's position being reconciled with the authoritative one.
//
// Unlike a plain lerp by rate*dt, this is frame-rate independent: halving the
// step size and taking twice as many lands in the same place, so a run at 30
// and one at 120 frames per second look alike.
func Approach[F Float](current, target, rate, dt F) F {
	return current + (target-current)*F(1-math.Exp(float64(-rate*dt)))
}
