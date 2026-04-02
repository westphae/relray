package lorentz

import (
	"math"
	"testing"

	"sif/gogs/eric/relray/pkg/vec"
)

const eps = 1e-9

func assertClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Errorf("%s: got %v, want %v", label, got, want)
	}
}

func TestGamma(t *testing.T) {
	assertClose(t, "beta=0", Gamma(vec.Vec3{}), 1.0)
	assertClose(t, "beta=0.5", Gamma(vec.Vec3{Z: 0.5}), 1.0/math.Sqrt(0.75))
	assertClose(t, "beta=0.9", Gamma(vec.Vec3{X: 0.9}), 1.0/math.Sqrt(1-0.81))
}

func TestAberrateForward(t *testing.T) {
	beta := vec.Vec3{Z: 0.5}
	r := Aberrate(vec.Vec3{Z: 1}, beta)
	assertClose(t, "forward dir.X", r.Dir.X, 0.0)
	assertClose(t, "forward dir.Y", r.Dir.Y, 0.0)
	assertClose(t, "forward dir.Z", r.Dir.Z, 1.0)
	// Looking forward while moving forward = blueshift
	// D = sqrt((1+beta)/(1-beta))
	assertClose(t, "forward Doppler", r.Doppler, math.Sqrt(3.0))
}

func TestAberrateBackward(t *testing.T) {
	beta := vec.Vec3{Z: 0.5}
	r := Aberrate(vec.Vec3{Z: -1}, beta)
	assertClose(t, "backward dir.Z", r.Dir.Z, -1.0)
	// Looking backward = receding = redshift
	assertClose(t, "backward Doppler", r.Doppler, 1.0/math.Sqrt(3.0))
}

func TestAberrateSideways(t *testing.T) {
	beta := vec.Vec3{Z: 0.5}
	r := Aberrate(vec.Vec3{X: 1}, beta)

	// Transverse Doppler = 1/gamma (always a redshift)
	gamma := Gamma(beta)
	assertClose(t, "sideways Doppler", r.Doppler, 1.0/gamma)

	// A sideways ray in the observer frame maps to a MORE BACKWARD direction
	// in the world frame. This is correct: aberration pulls world-frame objects
	// forward in the observer's perception, so what appears sideways to the
	// observer is actually behind them in the world frame.
	if r.Dir.Z >= 0 {
		t.Errorf("expected negative Z (world-frame backward) for sideways observer ray, got %v", r.Dir.Z)
	}
	if r.Dir.X <= 0 {
		t.Errorf("expected positive X preserved, got %v", r.Dir.X)
	}
	assertClose(t, "unit length", r.Dir.Length(), 1.0)
}

func TestAberrateZeroBeta(t *testing.T) {
	d := vec.Vec3{X: 0.5, Y: 0.5, Z: math.Sqrt(0.5)}.Normalize()
	r := Aberrate(d, vec.Vec3{})
	assertClose(t, "Doppler", r.Doppler, 1.0)
	assertClose(t, "dir.X", r.Dir.X, d.X)
	assertClose(t, "dir.Y", r.Dir.Y, d.Y)
	assertClose(t, "dir.Z", r.Dir.Z, d.Z)
}

func TestAberrateHighBeta(t *testing.T) {
	// At high beta, aberration is extreme. A ray that is slightly backward
	// in the observer frame is very backward in the world frame.
	beta := vec.Vec3{Z: 0.9}
	dirObs := vec.Vec3{X: 1, Z: -0.1}.Normalize()
	r := Aberrate(dirObs, beta)
	// World-frame direction should be strongly backward
	if r.Dir.Z >= dirObs.Z {
		t.Errorf("at beta=0.9, expected world dir more backward than observer dir, got dir.Z=%v", r.Dir.Z)
	}
}

func TestAberrateDopplerSymmetry(t *testing.T) {
	// D(forward) * D(backward) = 1
	beta := vec.Vec3{Z: 0.5}
	rf := Aberrate(vec.Vec3{Z: 1}, beta)
	rb := Aberrate(vec.Vec3{Z: -1}, beta)
	assertClose(t, "D_fwd * D_bwd", rf.Doppler*rb.Doppler, 1.0)
}

func TestAberrateRoundTrip(t *testing.T) {
	// Aberrating with +beta then -beta should recover the original direction.
	beta := vec.Vec3{X: 0.2, Z: 0.3}
	dirObs := vec.Vec3{X: 0.5, Y: 0.7, Z: -0.3}.Normalize()

	r1 := Aberrate(dirObs, beta)
	r2 := Aberrate(r1.Dir, beta.Neg())

	assertClose(t, "round-trip X", r2.Dir.X, dirObs.X)
	assertClose(t, "round-trip Y", r2.Dir.Y, dirObs.Y)
	assertClose(t, "round-trip Z", r2.Dir.Z, dirObs.Z)
	assertClose(t, "round-trip Doppler", r1.Doppler*r2.Doppler, 1.0)
}
