package retarded

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// C is the speed of light in scene units per second.
// In our slow-light universe, this is ~1 m/s.
const C = 1.0

// Trajectory describes the position of an object as a function of time.
type Trajectory func(t float64) vec.Vec3

// Solve finds the emission time t_emit for a photon that arrives at the
// observer position obsPos at time tObs, having been emitted by an object
// following the given trajectory.
//
// The constraint is: |obsPos - traj(t_emit)| = C * (tObs - t_emit)
// i.e., the light travel time equals the spatial distance divided by C.
//
// Uses Newton's method on f(t) = |obsPos - traj(t)|² - C²*(tObs - t)²
// Returns the emission time and the object's position at that time.
// Returns ok=false if the solver fails to converge.
func Solve(obsPos vec.Vec3, tObs float64, traj Trajectory) (tEmit float64, objPos vec.Vec3, ok bool) {
	// Initial guess: assume object is at its current position
	// Light travel time ≈ distance / C
	pos0 := traj(tObs)
	dist0 := obsPos.Sub(pos0).Length()
	tEmit = tObs - dist0/C

	const (
		maxIter = 50
		tol     = 1e-10
		dt      = 1e-8 // for numerical derivative of trajectory
	)

	for range maxIter {
		objPos = traj(tEmit)
		delta := obsPos.Sub(objPos)
		dist := delta.Length()
		timeDiff := tObs - tEmit

		// f(t) = dist² - C²*timeDiff²
		f := dist*dist - C*C*timeDiff*timeDiff

		if math.Abs(f) < tol {
			return tEmit, objPos, true
		}

		// f'(t) = d/dt[|obsPos - traj(t)|² - C²*(tObs - t)²]
		//       = -2*(obsPos - traj(t))·traj'(t) + 2*C²*(tObs - t)
		// Compute traj'(t) numerically
		posPlus := traj(tEmit + dt)
		vel := posPlus.Sub(objPos).Scale(1.0 / dt)

		fp := -2*delta.Dot(vel) + 2*C*C*timeDiff

		if math.Abs(fp) < 1e-20 {
			break // derivative too small, bail
		}

		tEmit -= f / fp

		// Sanity: t_emit must be before t_obs
		if tEmit > tObs {
			tEmit = tObs - 1e-6
		}
	}

	// Check if we converged close enough
	objPos = traj(tEmit)
	dist := obsPos.Sub(objPos).Length()
	timeDiff := tObs - tEmit
	residual := math.Abs(dist - C*timeDiff)
	if residual < 1e-6 {
		return tEmit, objPos, true
	}
	return 0, vec.Vec3{}, false
}
