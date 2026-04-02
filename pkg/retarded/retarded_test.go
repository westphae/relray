package retarded

import (
	"math"
	"testing"

	"sif/gogs/eric/relray/pkg/vec"
)

const eps = 1e-6

func TestSolveStationary(t *testing.T) {
	// Stationary object at (0, 0, 5). Observer at origin, tObs=10.
	// Light travel time = 5/C = 5s. So t_emit = 5.
	traj := func(t float64) vec.Vec3 { return vec.Vec3{Z: 5} }
	tEmit, pos, ok := Solve(vec.Vec3{}, 10.0, traj)
	if !ok {
		t.Fatal("solver failed")
	}
	if math.Abs(tEmit-5.0) > eps {
		t.Errorf("tEmit = %v, want 5.0", tEmit)
	}
	if math.Abs(pos.Z-5.0) > eps {
		t.Errorf("pos.Z = %v, want 5.0", pos.Z)
	}
}

func TestSolveMovingAway(t *testing.T) {
	// Object moving at 0.5c along +Z, starting at Z=0 at t=0.
	// traj(t) = (0, 0, 0.5*t)
	// Observer at origin, tObs=10.
	// Constraint: 0.5*t_emit = C*(10 - t_emit) = 10 - t_emit
	// 0.5*t_emit + t_emit = 10 → t_emit = 10/1.5 ≈ 6.667
	traj := func(t float64) vec.Vec3 { return vec.Vec3{Z: 0.5 * t} }
	tEmit, pos, ok := Solve(vec.Vec3{}, 10.0, traj)
	if !ok {
		t.Fatal("solver failed")
	}
	want := 10.0 / 1.5
	if math.Abs(tEmit-want) > eps {
		t.Errorf("tEmit = %v, want %v", tEmit, want)
	}
	if math.Abs(pos.Z-0.5*want) > eps {
		t.Errorf("pos.Z = %v, want %v", pos.Z, 0.5*want)
	}
}

func TestSolveMovingPerpendicular(t *testing.T) {
	// Object moving at 0.3c along +X, starting at (0, 0, 3) at t=0.
	// traj(t) = (0.3*t, 0, 3)
	// Observer at origin, tObs=10.
	// |(-0.3*te, 0, -3)| = 10 - te
	// sqrt(0.09*te² + 9) = 10 - te
	// 0.09*te² + 9 = 100 - 20*te + te²
	// 0 = 0.91*te² - 20*te + 91
	// te = (20 ± sqrt(400 - 4*0.91*91)) / (2*0.91)
	// = (20 ± sqrt(400 - 331.24)) / 1.82
	// = (20 ± sqrt(68.76)) / 1.82
	// = (20 ± 8.292) / 1.82
	// te = 15.55 or te = 6.43
	// We want te < tObs=10, so te ≈ 6.43
	traj := func(t float64) vec.Vec3 { return vec.Vec3{X: 0.3 * t, Z: 3} }
	tEmit, pos, ok := Solve(vec.Vec3{}, 10.0, traj)
	if !ok {
		t.Fatal("solver failed")
	}
	wantTE := (20.0 - math.Sqrt(68.76)) / 1.82
	if math.Abs(tEmit-wantTE) > 1e-3 {
		t.Errorf("tEmit = %v, want ~%v", tEmit, wantTE)
	}
	// Object should appear displaced in the direction of motion
	if pos.X <= 0 {
		t.Errorf("object should be at positive X, got %v", pos.X)
	}
}
