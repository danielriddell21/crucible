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
