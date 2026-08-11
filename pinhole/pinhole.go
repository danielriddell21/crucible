// Package pinhole is the perspective camera: it maps between world space and
// the image a lens sees, in both directions.
//
// It completes the family's set of camera models. [view] frames a 2D world
// orthographically, [raycast] walks columns through a 2.5D grid, and this
// projects a full 3D scene through a pinhole — the same projection a GPU
// applies, done in plain arithmetic so a headless harness can reproduce
// exactly what a rendered frame would show.
//
// The two directions serve different callers:
//
//   - Forward. A [View] is a lens placed in the world. [View.Project] maps a
//     point to a pixel and [View.BoxBounds] maps an oriented box to the
//     rectangle it covers, which is what an annotation or a hit test needs.
//   - Inverse. A [Rig] is a lens mounted a fixed height above flat ground and
//     tilted down. [Rig.GroundDistance] and [Rig.RangeAtHeight] recover how
//     far away something is from where it sits in the image, and [Lens.Bearing]
//     which way. This is the calibration a perception system reasons with when
//     all it has is pixels.
//
// Angles are radians unless a field says otherwise, distances are whatever
// unit the world uses, and pixel coordinates put the origin at the top left.
//
// [view]: https://pkg.go.dev/github.com/danielriddell21/crucible/view
// [raycast]: https://pkg.go.dev/github.com/danielriddell21/crucible/raycast
package pinhole

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
)

// Lens is the image geometry alone: how big the picture is and how much of the
// world fits into it.
type Lens struct {
	// Width and Height are the image size in pixels.
	Width, Height int
	// FovY is the vertical field of view in degrees. The horizontal field
	// follows from the aspect ratio, which is what a GPU projection does too,
	// so a software projection and a rendered frame agree.
	FovY float64
}

// Focal returns the focal length in pixels: the distance from the pinhole to
// the image plane, in the same units the image is measured in. It is the
// constant behind every projection here.
func (l Lens) Focal() float64 {
	return float64(l.Height) / 2 / math.Tan(l.FovY*0.5*math.Pi/180)
}

// Bearing returns the horizontal angle from straight ahead to an image column,
// positive to the right.
func (l Lens) Bearing(pixelX float64) float64 {
	return math.Atan2(pixelX-float64(l.Width)/2, l.Focal())
}

// Elevation returns the vertical angle from the optical axis to an image row,
// positive downward — image rows increase downward, and so does this.
func (l Lens) Elevation(pixelY float64) float64 {
	return math.Atan2(pixelY-float64(l.Height)/2, l.Focal())
}

// View is a [Lens] placed in the world, looking from Position toward Target.
type View struct {
	Lens
	Position, Target geom.Vec3
}

// basis is the view's camera-space axes and eye point, worked out once so
// projecting many points does not redo it.
type basis struct {
	eye                 geom.Vec3
	right, up, fwd      geom.Vec3
	focal, halfW, halfH float64
}

func (v View) basis() basis {
	fwd := v.Target.Sub(v.Position).Normalize()
	// A camera looking straight up or down has no usable roll reference, so
	// pick a different world up for that case.
	worldUp := geom.Vec3{Y: 1}
	if math.Abs(fwd.Dot(worldUp)) > 0.999 {
		worldUp = geom.Vec3{Z: 1}
	}
	right := fwd.Cross(worldUp).Normalize()
	return basis{
		eye: v.Position, fwd: fwd, right: right, up: right.Cross(fwd).Normalize(),
		focal: v.Focal(),
		halfW: float64(v.Width) / 2, halfH: float64(v.Height) / 2,
	}
}

// nearPlane is the camera-space depth below which a point is treated as behind
// the lens. Projecting anything closer divides by roughly zero and throws the
// result across the image.
const nearPlane = 0.05

func (b basis) project(p geom.Vec3) (geom.Vec2, bool) {
	rel := p.Sub(b.eye)
	z := rel.Dot(b.fwd)
	if z <= nearPlane {
		return geom.Vec2{}, false
	}
	return geom.Vec2{
		X: b.halfW + b.focal*rel.Dot(b.right)/z,
		Y: b.halfH - b.focal*rel.Dot(b.up)/z,
	}, true
}

// Project maps a world point to its pixel position. It reports false for a
// point at or behind the lens, whose projection is meaningless — check the
// result before using it, or a point behind the camera will appear in front of
// it, mirrored.
//
// The returned position may lie outside the image; clip it if that matters.
func (v View) Project(p geom.Vec3) (geom.Vec2, bool) {
	return v.basis().project(p)
}

// Box is an oriented box standing in the world: a car, a crate, a sign.
type Box struct {
	// Center is the box's middle, including its height off the ground.
	Center geom.Vec3
	// HalfW, HalfH and HalfL are its half extents across, up, and along its
	// facing.
	HalfW, HalfH, HalfL float64
	// Yaw rotates the box about the vertical axis, in radians, measured so
	// that a yaw of zero faces along +X.
	Yaw float64
}

// Corners returns the box's eight corners in world space.
func (b Box) Corners() [8]geom.Vec3 {
	fwd := geom.Vec2{X: math.Cos(b.Yaw), Y: math.Sin(b.Yaw)}
	right := geom.Vec2{X: -fwd.Y, Y: fwd.X}

	var out [8]geom.Vec3
	for i := range out {
		// The three bits of i pick the sign of each half extent.
		sx, sy, sz := sign(i, 1), sign(i, 2), sign(i, 4)
		off := fwd.Scale(b.HalfL * sz).Add(right.Scale(b.HalfW * sx))
		out[i] = geom.Vec3{
			X: b.Center.X + off.X,
			Y: b.Center.Y + b.HalfH*sy,
			Z: b.Center.Z + off.Y,
		}
	}
	return out
}

func sign(i, bit int) float64 {
	if i&bit == 0 {
		return -1
	}
	return 1
}

// BoxBounds returns the axis-aligned rectangle in image space that a box's
// projection covers, as the corners of a [geom.Vec2] pair: min is the top-left
// and max the bottom-right.
//
// Corners behind the lens are skipped, so a box the camera is inside still
// yields the bounds of the part in front. It reports false only when no corner
// is in front at all. The bounds are not clipped to the image — a caller that
// wants a drawable rectangle clips them itself, and one that wants to know how
// big the box would be, including the part off-screen, does not.
func (v View) BoxBounds(b Box) (minPt, maxPt geom.Vec2, ok bool) {
	bs := v.basis()
	minPt = geom.Vec2{X: math.Inf(1), Y: math.Inf(1)}
	maxPt = geom.Vec2{X: math.Inf(-1), Y: math.Inf(-1)}

	for _, c := range b.Corners() {
		p, in := bs.project(c)
		if !in {
			continue
		}
		ok = true
		minPt.X, minPt.Y = math.Min(minPt.X, p.X), math.Min(minPt.Y, p.Y)
		maxPt.X, maxPt.Y = math.Max(maxPt.X, p.X), math.Max(maxPt.Y, p.Y)
	}
	if !ok {
		return geom.Vec2{}, geom.Vec2{}, false
	}
	return minPt, maxPt, true
}

// Rig is a [Lens] mounted a fixed height above flat ground and tilted
// downward: a camera on a bonnet, a mast, a drone holding station. Where a
// [View] answers "where does this world point land in the image", a Rig
// answers the harder question the other way round — "how far away is the thing
// I can see at this row" — which has an answer only because the mounting
// geometry is known.
type Rig struct {
	Lens
	// Height is how far the lens sits above the ground plane.
	Height float64
	// Pitch is the downward tilt in radians. Zero looks at the horizon.
	Pitch float64
}

// Unreachable is returned by the ranging methods for an image row whose ray
// never meets the height asked about — at or above the horizon, looking for
// the ground. It is deliberately a large finite distance rather than an
// infinity, so arithmetic on the result stays well behaved and a caller that
// forgets to check merely believes the thing is very far away.
const Unreachable = 1e5

// horizonEpsilon is how close to level a ray may get before its intersection
// with the ground is treated as unreachable. Rays flatter than this produce
// distances so large and so sensitive to a pixel of noise that they are not
// worth reporting.
const horizonEpsilon = 0.012

// GroundDistance estimates how far ahead the ground under an image row lies,
// assuming the ground is flat. A row at or above the horizon returns
// [Unreachable].
//
// This is the ranging a perception system does on anything standing on the
// ground: find where an object meets the road in the image, and its distance
// follows from the geometry.
func (r Rig) GroundDistance(pixelY float64) float64 {
	down := r.Pitch + r.Elevation(pixelY)
	if down <= horizonEpsilon {
		return Unreachable
	}
	return r.Height / math.Tan(down)
}

// RangeAtHeight estimates how far ahead a point lies when its height above the
// ground is known, which is what lets something mounted in the air — a traffic
// signal, a sign, a lamp — be ranged directly rather than by finding where it
// meets the ground.
//
// [Rig.GroundDistance] is the special case of this at height zero. A point
// above the lens appears above the optical axis, so the sign of the drop and
// of the ray angle agree and the result stays positive. It returns
// [Unreachable] when the ray never reaches that height ahead of the camera.
func (r Rig) RangeAtHeight(pixelY, height float64) float64 {
	down := r.Pitch + r.Elevation(pixelY)
	drop := r.Height - height
	t := math.Tan(down)
	if math.Abs(t) < 1e-4 || drop == 0 {
		return Unreachable
	}
	if d := drop / t; d > 0 {
		return d
	}
	return Unreachable
}
