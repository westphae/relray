package vec

import (
	"math"
	"testing"
)

func TestIdentityMulVec(t *testing.T) {
	v := Vec3{1, 2, 3}
	got := Identity().MulVec(v)
	assertVec(t, "identity", got, v)
}

func TestRotationYRoundTrip(t *testing.T) {
	r := RotationY(math.Pi / 4)
	v := Vec3{1, 0, 0}
	rotated := r.MulVec(v)
	recovered := r.Transpose().MulVec(rotated)
	assertVec(t, "round-trip", recovered, v)
}

func TestRotationY90(t *testing.T) {
	r := RotationY(math.Pi / 2)
	v := Vec3{1, 0, 0}
	got := r.MulVec(v)
	assertVec(t, "Ry(90)*X", got, Vec3{0, 0, -1})
}

func TestRotationX90(t *testing.T) {
	r := RotationX(math.Pi / 2)
	v := Vec3{0, 1, 0}
	got := r.MulVec(v)
	assertVec(t, "Rx(90)*Y", got, Vec3{0, 0, 1})
}

func TestRotationFromEulerDegIdentity(t *testing.T) {
	r := RotationFromEulerDeg(0, 0, 0)
	v := Vec3{1, 2, 3}
	got := r.MulVec(v)
	assertVec(t, "euler(0,0,0)", got, v)
}

func TestRotationFromAxisAngle(t *testing.T) {
	// 90 degrees around Y should map X → -Z
	r := RotationFromAxisAngle(Vec3{0, 1, 0}, math.Pi/2)
	got := r.MulVec(Vec3{1, 0, 0})
	assertVec(t, "axis-angle Y 90", got, Vec3{0, 0, -1})
}

func TestRotationOrthogonal(t *testing.T) {
	r := RotationFromEulerDeg(30, 45, 60)
	// R * R^T should be identity
	product := r.Mul(r.Transpose())
	id := Identity()
	for i := 0; i < 3; i++ {
		assertVec(t, "orthogonal row", Vec3{product[i].X, product[i].Y, product[i].Z},
			Vec3{id[i].X, id[i].Y, id[i].Z})
	}
}
