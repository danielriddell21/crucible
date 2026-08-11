package geom_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/geom"
)

const eps = 1e-9

func almost(a, b float64) bool { return math.Abs(a-b) < eps }

func TestVecArithmetic(t *testing.T) {
	a := geom.Vec2{X: 1, Y: 2}
	b := geom.Vec2{X: 3, Y: -1}
	if got := a.Add(b); got != (geom.Vec2{X: 4, Y: 1}) {
		t.Errorf("Add = %v", got)
	}
	if got := a.Sub(b); got != (geom.Vec2{X: -2, Y: 3}) {
		t.Errorf("Sub = %v", got)
	}
	if got := a.Scale(2); got != (geom.Vec2{X: 2, Y: 4}) {
		t.Errorf("Scale = %v", got)
	}
	if got := a.Dot(b); !almost(got, 1) {
		t.Errorf("Dot = %v", got)
	}
}

func TestLenDist(t *testing.T) {
	v := geom.Vec2{X: 3, Y: 4}
	if !almost(v.Len(), 5) {
		t.Errorf("Len = %v", v.Len())
	}
	o := geom.Vec2{X: 6, Y: 8}
	if !almost(v.Dist(o), 5) {
		t.Errorf("Dist = %v", v.Dist(o))
	}
	if !almost(v.DistSq(o), 25) {
		t.Errorf("DistSq = %v", v.DistSq(o))
	}
}

func TestNormalize(t *testing.T) {
	v := geom.Vec2{X: 3, Y: 4}.Normalize()
	if !almost(v.Len(), 1) {
		t.Errorf("normalized length = %v", v.Len())
	}
	zero := geom.Vec2{}
	if zero.Normalize() != zero {
		t.Errorf("zero vector must normalize to itself")
	}
}

func TestAngleRoundTrip(t *testing.T) {
	for _, a := range []float64{0, 1, -2, math.Pi / 3} {
		v := geom.FromAngle(a)
		if !almost(v.Len(), 1) {
			t.Errorf("FromAngle(%v) not unit", a)
		}
		if !almost(v.Angle(), a) {
			t.Errorf("Angle round trip: got %v want %v", v.Angle(), a)
		}
	}
}

func TestToroidal(t *testing.T) {
	v := geom.Vec2{X: 9, Y: 9}
	if got := v.WrapTo(10, 10); got != v {
		t.Errorf("WrapTo inside = %v", got)
	}
	if got := (geom.Vec2{X: -1, Y: 11}).WrapTo(10, 10); got != (geom.Vec2{X: 9, Y: 1}) {
		t.Errorf("WrapTo outside = %v", got)
	}
	// Crossing the seam is shorter than going the long way round.
	a, b := geom.Vec2{X: 1, Y: 5}, geom.Vec2{X: 9, Y: 5}
	if d := a.ToroidalDist(b, 10, 10); !almost(d, 2) {
		t.Errorf("ToroidalDist = %v", d)
	}
	if got := a.ShortestDelta(b, 10, 10); !almost(got.X, -2) || !almost(got.Y, 0) {
		t.Errorf("ShortestDelta = %v", got)
	}
}

func TestCoord(t *testing.T) {
	c := geom.Coord{X: 2, Y: 3}
	if got := c.Add(geom.Coord{X: 1, Y: -1}); got != (geom.Coord{X: 3, Y: 2}) {
		t.Errorf("Add = %v", got)
	}
	if got := c.Vec2(); got != (geom.Vec2{X: 2.5, Y: 3.5}) {
		t.Errorf("Vec2 = %v", got)
	}
}

func TestRect(t *testing.T) {
	r := geom.Rect{X: 2, Y: 4, W: 6, H: 8}
	if got := r.Center(); got != (geom.Coord{X: 5, Y: 8}) {
		t.Errorf("Center = %v", got)
	}
	if !r.Contains(geom.Coord{X: 2, Y: 4}) || r.Contains(geom.Coord{X: 8, Y: 4}) {
		t.Errorf("Contains: wrong bounds")
	}
}

func TestClamp(t *testing.T) {
	if got := geom.Clamp(5, 0, 3); got != 3 {
		t.Errorf("Clamp high = %v", got)
	}
	if got := geom.Clamp(-1, 0, 3); got != 0 {
		t.Errorf("Clamp low = %v", got)
	}
	if got := geom.Clamp(2, 0, 3); got != 2 {
		t.Errorf("Clamp mid = %v", got)
	}
	if got := geom.Clamp01(1.5); got != 1 {
		t.Errorf("Clamp01 = %v", got)
	}
}

func TestWrapPi(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0},
		{math.Pi / 2, math.Pi / 2},
		{math.Pi, math.Pi},
		{-math.Pi, -math.Pi},
		{3 * math.Pi / 2, -math.Pi / 2},
		{-3 * math.Pi / 2, math.Pi / 2},
		{7 * math.Pi, math.Pi},
		{-7 * math.Pi, -math.Pi},
	}
	for _, c := range cases {
		if got := geom.WrapPi(c.in); math.Abs(got-c.want) > eps {
			t.Errorf("WrapPi(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestWrapPiStaysInRange(t *testing.T) {
	// Every multiple of a third of a turn, out to ten turns either way.
	for i := -60; i <= 60; i++ {
		a := float64(i) * (2 * math.Pi / 3)
		if got := geom.WrapPi(a); got < -math.Pi || got > math.Pi {
			t.Fatalf("WrapPi(%v) = %v, outside [-pi, pi]", a, got)
		}
	}
}

func TestWrapPiKeepsFloat32(t *testing.T) {
	// The point of the generic: a float32 caller gets a float32 back rather
	// than casting through float64 at the call site.
	var a float32 = 3 * math.Pi / 2
	got := geom.WrapPi(a)
	if want := float32(-math.Pi / 2); math.Abs(float64(got-want)) > 1e-6 {
		t.Errorf("WrapPi(%v) = %v, want %v", a, got, want)
	}
}

func TestAngleDiff(t *testing.T) {
	cases := []struct{ a, b, want float64 }{
		{0, 0, 0},
		{math.Pi / 4, -math.Pi / 4, math.Pi / 2},
		// Across the wrap: just past -pi is a short hop from just under pi.
		{-3 * math.Pi / 4, 3 * math.Pi / 4, math.Pi / 2},
		{3 * math.Pi / 4, -3 * math.Pi / 4, -math.Pi / 2},
	}
	for _, c := range cases {
		if got := geom.AngleDiff(c.a, c.b); math.Abs(got-c.want) > eps {
			t.Errorf("AngleDiff(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestVec3Arithmetic(t *testing.T) {
	a := geom.Vec3{X: 1, Y: 2, Z: 3}
	b := geom.Vec3{X: 4, Y: 5, Z: 6}

	if got := a.Add(b); got != (geom.Vec3{X: 5, Y: 7, Z: 9}) {
		t.Errorf("Add = %v", got)
	}
	if got := b.Sub(a); got != (geom.Vec3{X: 3, Y: 3, Z: 3}) {
		t.Errorf("Sub = %v", got)
	}
	if got := a.Scale(2); got != (geom.Vec3{X: 2, Y: 4, Z: 6}) {
		t.Errorf("Scale = %v", got)
	}
	if got := a.Dot(b); got != 32 {
		t.Errorf("Dot = %v, want 32", got)
	}
	if got := a.Ground(); got != (geom.Vec2{X: 1, Y: 3}) {
		t.Errorf("Ground = %v, want the XZ components", got)
	}
}

func TestVec3Cross(t *testing.T) {
	x := geom.Vec3{X: 1}
	y := geom.Vec3{Y: 1}
	// Right-handed: x cross y is +z.
	if got := x.Cross(y); got != (geom.Vec3{Z: 1}) {
		t.Errorf("x cross y = %v, want +z", got)
	}
	if got := y.Cross(x); got != (geom.Vec3{Z: -1}) {
		t.Errorf("y cross x = %v, want -z", got)
	}
	if got := x.Cross(x); got != (geom.Vec3{}) {
		t.Errorf("x cross x = %v, want the zero vector", got)
	}
}

func TestVec3LenAndNormalize(t *testing.T) {
	v := geom.Vec3{X: 2, Y: 3, Z: 6}
	if got := v.Len(); got != 7 {
		t.Errorf("Len = %v, want 7", got)
	}
	if got := (geom.Vec3{X: 1}).Dist(geom.Vec3{X: 4, Y: 4}); got != 5 {
		t.Errorf("Dist = %v, want 5", got)
	}
	if got := v.Normalize().Len(); math.Abs(got-1) > eps {
		t.Errorf("Normalize().Len() = %v, want 1", got)
	}
	if got := (geom.Vec3{}).Normalize(); got != (geom.Vec3{}) {
		t.Errorf("zero Normalize = %v, want the zero vector", got)
	}
}

func TestMoveToward(t *testing.T) {
	if got := geom.MoveToward(0.0, 10.0, 3); got != 3 {
		t.Errorf("up = %v, want 3", got)
	}
	if got := geom.MoveToward(0.0, -10.0, 3); got != -3 {
		t.Errorf("down = %v, want -3", got)
	}
	// It lands on the target rather than stepping past it.
	if got := geom.MoveToward(0.0, 2.0, 5); got != 2 {
		t.Errorf("overshoot = %v, want 2", got)
	}
	if got := geom.MoveToward(0.0, -2.0, 5); got != -2 {
		t.Errorf("negative overshoot = %v, want -2", got)
	}
	if got := geom.MoveToward(4.0, 4.0, 1); got != 4 {
		t.Errorf("already there = %v, want 4", got)
	}
}

func TestApproachIsFrameRateIndependent(t *testing.T) {
	// One second of easing must land in the same place however it is sliced.
	const rate = 4.0
	coarse := 0.0
	for range 30 {
		coarse = geom.Approach(coarse, 100.0, rate, 1.0/30)
	}
	fine := 0.0
	for range 120 {
		fine = geom.Approach(fine, 100.0, rate, 1.0/120)
	}
	if math.Abs(coarse-fine) > 1e-9 {
		t.Errorf("30fps reached %v but 120fps reached %v", coarse, fine)
	}
}

func TestApproachConverges(t *testing.T) {
	v := 0.0
	for range 500 {
		v = geom.Approach(v, 7.0, 5, 1.0/60)
	}
	if math.Abs(v-7) > 1e-6 {
		t.Errorf("converged to %v, want 7", v)
	}
	// A zero step leaves the value alone.
	if got := geom.Approach(3.0, 9.0, 5, 0); got != 3 {
		t.Errorf("zero dt moved the value to %v", got)
	}
}
