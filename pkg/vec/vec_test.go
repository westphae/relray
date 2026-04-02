package vec

import (
	"math"
	"testing"
)

const eps = 1e-12

func assertClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Errorf("%s: got %v, want %v", label, got, want)
	}
}

func assertVec(t *testing.T, label string, got, want Vec3) {
	t.Helper()
	assertClose(t, label+".X", got.X, want.X)
	assertClose(t, label+".Y", got.Y, want.Y)
	assertClose(t, label+".Z", got.Z, want.Z)
}

func TestBasicOps(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}
	assertVec(t, "Add", a.Add(b), Vec3{5, 7, 9})
	assertVec(t, "Sub", a.Sub(b), Vec3{-3, -3, -3})
	assertVec(t, "Scale", a.Scale(2), Vec3{2, 4, 6})
	assertClose(t, "Dot", a.Dot(b), 32)
	assertVec(t, "Cross", a.Cross(b), Vec3{-3, 6, -3})
	assertVec(t, "Neg", a.Neg(), Vec3{-1, -2, -3})
}

func TestLength(t *testing.T) {
	a := Vec3{3, 4, 0}
	assertClose(t, "Length", a.Length(), 5)
	assertClose(t, "LengthSq", a.LengthSq(), 25)
}

func TestNormalize(t *testing.T) {
	a := Vec3{0, 3, 4}
	n := a.Normalize()
	assertClose(t, "normalized length", n.Length(), 1)
	assertVec(t, "Normalize", n, Vec3{0, 0.6, 0.8})

	zero := Vec3{}
	assertVec(t, "zero normalize", zero.Normalize(), Vec3{})
}

func TestReflect(t *testing.T) {
	// Ray going down-right, reflected off horizontal surface (normal pointing up)
	d := Vec3{1, -1, 0}.Normalize()
	n := Vec3{0, 1, 0}
	r := d.Reflect(n)
	want := Vec3{1, 1, 0}.Normalize()
	assertVec(t, "Reflect", r, want)
}
