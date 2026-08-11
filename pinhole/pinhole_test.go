package pinhole_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/pinhole"
)

const eps = 1e-6

// lens is a plain 640x360 camera with a 60 degree vertical field of view.
var lens = pinhole.Lens{Width: 640, Height: 360, FovY: 60}

// looking builds a view at the origin, one unit up, facing along +X.
func looking() pinhole.View {
	return pinhole.View{
		Lens:     lens,
		Position: geom.Vec3{Y: 1},
		Target:   geom.Vec3{X: 1, Y: 1},
	}
}

func nearly(t *testing.T, what string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-3 {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestFocalMatchesTheFieldOfView(t *testing.T) {
	// Half the image height subtends half the vertical field of view.
	f := lens.Focal()
	got := 2 * math.Atan(float64(lens.Height)/2/f) * 180 / math.Pi
	nearly(t, "recovered FovY", got, lens.FovY)
}

func TestProjectPutsTheCentreOfViewInTheCentreOfTheImage(t *testing.T) {
	v := looking()
	p, ok := v.Project(geom.Vec3{X: 25, Y: 1})
	if !ok {
		t.Fatal("a point straight ahead did not project")
	}
	nearly(t, "x", p.X, 320)
	nearly(t, "y", p.Y, 180)
}

func TestProjectRightAndUp(t *testing.T) {
	v := looking()
	// In a Y-up, right-handed world with the camera facing +X, the camera's
	// right is +Z.
	right, ok := v.Project(geom.Vec3{X: 10, Y: 1, Z: 2})
	if !ok {
		t.Fatal("point did not project")
	}
	if right.X <= 320 {
		t.Errorf("a point at +Z landed at x=%v, want right of centre", right.X)
	}
	above, ok := v.Project(geom.Vec3{X: 10, Y: 3})
	if !ok {
		t.Fatal("point did not project")
	}
	if above.Y >= 180 {
		t.Errorf("a point above the lens landed at y=%v, want above centre", above.Y)
	}
}

func TestProjectionScalesWithDistance(t *testing.T) {
	// Twice as far away, half as far from the centre of the image.
	v := looking()
	near, _ := v.Project(geom.Vec3{X: 10, Y: 1, Z: 1})
	far, _ := v.Project(geom.Vec3{X: 20, Y: 1, Z: 1})
	nearly(t, "far offset", far.X-320, (near.X-320)/2)
}

func TestProjectRejectsPointsBehindTheLens(t *testing.T) {
	v := looking()
	for _, p := range []geom.Vec3{
		{X: -5, Y: 1},   // behind
		{X: 0, Y: 1},    // exactly at the lens
		{X: 0.01, Y: 1}, // inside the near plane
	} {
		if _, ok := v.Project(p); ok {
			t.Errorf("point %v projected, want it rejected", p)
		}
	}
}

func TestProjectLookingStraightDown(t *testing.T) {
	// A camera with no roll reference must still produce a usable basis
	// rather than dividing by a zero-length cross product.
	v := pinhole.View{Lens: lens, Position: geom.Vec3{Y: 50}, Target: geom.Vec3{}}
	p, ok := v.Project(geom.Vec3{})
	if !ok {
		t.Fatal("the point under a top-down camera did not project")
	}
	nearly(t, "x", p.X, 320)
	nearly(t, "y", p.Y, 180)
}

func TestBoxCorners(t *testing.T) {
	b := pinhole.Box{
		Center: geom.Vec3{X: 10, Y: 1, Z: 0},
		HalfW:  1, HalfH: 0.5, HalfL: 2,
	}
	cs := b.Corners()

	minX, maxX, minY, maxY, minZ, maxZ := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, c := range cs {
		minX, maxX = math.Min(minX, c.X), math.Max(maxX, c.X)
		minY, maxY = math.Min(minY, c.Y), math.Max(maxY, c.Y)
		minZ, maxZ = math.Min(minZ, c.Z), math.Max(maxZ, c.Z)
	}
	// Yaw zero faces +X, so HalfL runs along X and HalfW across Z.
	nearly(t, "X span", maxX-minX, 4)
	nearly(t, "Y span", maxY-minY, 1)
	nearly(t, "Z span", maxZ-minZ, 2)
}

func TestBoxCornersRotate(t *testing.T) {
	// A quarter turn swaps which axis the length runs along.
	b := pinhole.Box{Center: geom.Vec3{}, HalfW: 1, HalfH: 0.5, HalfL: 2, Yaw: math.Pi / 2}
	minX, maxX, minZ, maxZ := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, c := range b.Corners() {
		minX, maxX = math.Min(minX, c.X), math.Max(maxX, c.X)
		minZ, maxZ = math.Min(minZ, c.Z), math.Max(maxZ, c.Z)
	}
	nearly(t, "X span", maxX-minX, 2)
	nearly(t, "Z span", maxZ-minZ, 4)
}

func TestBoxBounds(t *testing.T) {
	v := looking()
	b := pinhole.Box{Center: geom.Vec3{X: 20, Y: 1}, HalfW: 1, HalfH: 1, HalfL: 2}

	lo, hi, ok := v.BoxBounds(b)
	if !ok {
		t.Fatal("a box straight ahead produced no bounds")
	}
	if lo.X >= hi.X || lo.Y >= hi.Y {
		t.Fatalf("bounds are inside out: %v to %v", lo, hi)
	}
	// It is centred on the image, and symmetric about the centre because the
	// box is centred on the optical axis.
	nearly(t, "horizontal centre", (lo.X+hi.X)/2, 320)
	nearly(t, "vertical centre", (lo.Y+hi.Y)/2, 180)
}

func TestBoxBoundsGrowAsItApproaches(t *testing.T) {
	v := looking()
	b := pinhole.Box{HalfW: 1, HalfH: 1, HalfL: 1, Center: geom.Vec3{X: 40, Y: 1}}
	farLo, farHi, _ := v.BoxBounds(b)

	b.Center.X = 10
	nearLo, nearHi, _ := v.BoxBounds(b)

	if (nearHi.X - nearLo.X) <= (farHi.X - farLo.X) {
		t.Errorf("a nearer box was not wider: %v vs %v", nearHi.X-nearLo.X, farHi.X-farLo.X)
	}
}

func TestBoxBoundsAreNotClippedToTheImage(t *testing.T) {
	// A caller sizing a detection wants the true extent, including the part
	// off-screen; one that wants to draw it clips for itself.
	v := looking()
	b := pinhole.Box{Center: geom.Vec3{X: 2, Y: 1}, HalfW: 20, HalfH: 20, HalfL: 1}

	lo, hi, ok := v.BoxBounds(b)
	if !ok {
		t.Fatal("no bounds")
	}
	if lo.X >= 0 && hi.X <= float64(v.Width) {
		t.Errorf("bounds %v to %v fit the image; the test is not exercising clipping", lo, hi)
	}
}

func TestBoxBoundsWithTheCameraInsideIt(t *testing.T) {
	// Some corners are behind the lens. The visible part still has bounds.
	v := looking()
	b := pinhole.Box{Center: geom.Vec3{Y: 1}, HalfW: 3, HalfH: 3, HalfL: 3}

	if _, _, ok := v.BoxBounds(b); !ok {
		t.Error("a box the camera stands inside produced no bounds")
	}
}

func TestBoxBoundsEntirelyBehind(t *testing.T) {
	v := looking()
	b := pinhole.Box{Center: geom.Vec3{X: -20, Y: 1}, HalfW: 1, HalfH: 1, HalfL: 1}
	if _, _, ok := v.BoxBounds(b); ok {
		t.Error("a box behind the camera produced bounds")
	}
}

// rig is a camera 1.35 units up, tilted slightly down — a bonnet mounting.
var rig = pinhole.Rig{Lens: lens, Height: 1.35, Pitch: 0.10}

func TestGroundDistanceFallsWithTheRow(t *testing.T) {
	// Further down the image is closer to the camera.
	last := math.Inf(1)
	for y := 200.0; y <= 350; y += 25 {
		d := rig.GroundDistance(y)
		if d >= last {
			t.Errorf("row %v is %v away, not nearer than the row above (%v)", y, d, last)
		}
		last = d
	}
}

func TestGroundDistanceAboveTheHorizonIsUnreachable(t *testing.T) {
	// Level and above: the ray never meets the ground ahead.
	if got := rig.GroundDistance(0); got != pinhole.Unreachable {
		t.Errorf("top of the image = %v, want Unreachable", got)
	}
	// The horizon sits above the image centre because the camera is tilted
	// down, so a row a little above centre is already past it.
	if got := rig.GroundDistance(150); got != pinhole.Unreachable {
		t.Errorf("a row above the horizon = %v, want Unreachable", got)
	}
}

func TestGroundDistanceIsTheRoundTripOfProjection(t *testing.T) {
	// Put a point on the ground a known distance ahead, project it, then
	// range the row it landed on. The two must agree — this is the property
	// that makes the inverse trustworthy.
	v := pinhole.View{
		Lens:     lens,
		Position: geom.Vec3{Y: rig.Height},
		// Tilted down by the rig's pitch: drop the target by tan(pitch) per
		// unit of distance.
		Target: geom.Vec3{X: 1, Y: rig.Height - math.Tan(rig.Pitch)},
	}
	for _, want := range []float64{8, 15, 30, 60} {
		p, ok := v.Project(geom.Vec3{X: want})
		if !ok {
			t.Fatalf("ground point at %v did not project", want)
		}
		nearly(t, "ranged distance", rig.GroundDistance(p.Y), want)
	}
}

func TestRangeAtHeightIsGroundDistanceAtZero(t *testing.T) {
	for _, y := range []float64{200, 250, 300} {
		nearly(t, "range at height 0", rig.RangeAtHeight(y, 0), rig.GroundDistance(y))
	}
}

func TestRangeAtHeightForSomethingMountedAboveTheLens(t *testing.T) {
	// A signal head 2 units up appears above the optical axis and still
	// ranges positively.
	v := pinhole.View{
		Lens:     lens,
		Position: geom.Vec3{Y: rig.Height},
		Target:   geom.Vec3{X: 1, Y: rig.Height - math.Tan(rig.Pitch)},
	}
	const height, want = 2.0, 30.0
	p, ok := v.Project(geom.Vec3{X: want, Y: height})
	if !ok {
		t.Fatal("the signal head did not project")
	}
	if p.Y >= 180 {
		t.Errorf("a head above the lens landed at y=%v, want above centre", p.Y)
	}
	nearly(t, "ranged distance", rig.RangeAtHeight(p.Y, height), want)
}

func TestRangeAtHeightAtTheLensHeightIsUnreachable(t *testing.T) {
	// A point level with the lens is on the optical plane: no drop, so no
	// intersection to solve for.
	if got := rig.RangeAtHeight(200, rig.Height); got != pinhole.Unreachable {
		t.Errorf("range at the lens height = %v, want Unreachable", got)
	}
}

func TestRangeAtHeightBehindIsUnreachable(t *testing.T) {
	// Looking down at a row well below centre, for something mounted above
	// the lens: that ray goes away from it, never toward it.
	if got := rig.RangeAtHeight(350, 4); got != pinhole.Unreachable {
		t.Errorf("range = %v, want Unreachable", got)
	}
}

func TestBearing(t *testing.T) {
	if got := lens.Bearing(float64(lens.Width) / 2); math.Abs(got) > eps {
		t.Errorf("centre bearing = %v, want 0", got)
	}
	right := lens.Bearing(float64(lens.Width) - 1)
	left := lens.Bearing(0)
	if right <= 0 || left >= 0 {
		t.Errorf("bearings do not straddle zero: left %v, right %v", left, right)
	}
	// Equal offsets either side of centre give equal and opposite angles.
	const off = 100
	centre := float64(lens.Width) / 2
	nearly(t, "symmetric", lens.Bearing(centre+off), -lens.Bearing(centre-off))
}

func TestBearingMatchesProjection(t *testing.T) {
	// Project a point off to one side, then recover its bearing from the
	// column it landed in.
	v := looking()
	const dist, across = 20.0, 6.0
	p, ok := v.Project(geom.Vec3{X: dist, Y: 1, Z: across})
	if !ok {
		t.Fatal("point did not project")
	}
	nearly(t, "bearing", lens.Bearing(p.X), math.Atan2(across, dist))
}

func TestElevation(t *testing.T) {
	if got := lens.Elevation(float64(lens.Height) / 2); math.Abs(got) > eps {
		t.Errorf("centre elevation = %v, want 0", got)
	}
	// Rows increase downward, and so does the angle.
	if got := lens.Elevation(float64(lens.Height) - 1); got <= 0 {
		t.Errorf("bottom elevation = %v, want positive (downward)", got)
	}
	if got := lens.Elevation(0); got >= 0 {
		t.Errorf("top elevation = %v, want negative (upward)", got)
	}
}
