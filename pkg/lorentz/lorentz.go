package lorentz

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Gamma returns the Lorentz factor for velocity beta (in units of c).
func Gamma(beta vec.Vec3) float64 {
	b2 := beta.LengthSq()
	if b2 == 0 {
		return 1.0
	}
	return 1.0 / math.Sqrt(1.0-b2)
}

// AberrationResult holds the world-frame ray direction and the Doppler factor,
// both computed from a single Lorentz boost of a null 4-vector.
type AberrationResult struct {
	Dir     vec.Vec3 // unit ray direction in world frame (camera → scene)
	Doppler float64  // frequency ratio f_obs / f_emit (>1 = blueshift)
}

// Aberrate transforms a ray direction from the observer's rest frame to the
// world frame, and simultaneously computes the Doppler factor.
//
// dirObs is the ray direction (camera → scene) in the observer's rest frame.
// beta is the observer's velocity as a fraction of c in the world frame.
//
// Method: the photon arriving at the camera propagates in direction -dirObs.
// Construct the photon's null 4-wavevector k_obs = (1, -dirObs) in the observer
// frame, then apply the Lorentz boost (+beta) to get k_world. The world-frame
// ray direction is the negation of the spatial part of k_world (normalized).
// The Doppler factor is f_obs/f_emit = 1/k_world^0.
func Aberrate(dirObs vec.Vec3, beta vec.Vec3) AberrationResult {
	b2 := beta.LengthSq()
	if b2 == 0 {
		return AberrationResult{Dir: dirObs, Doppler: 1.0}
	}

	gamma := 1.0 / math.Sqrt(1.0-b2)

	// Photon propagation direction in observer frame
	p := dirObs.Neg() // photon travels opposite to ray direction

	// Null 4-wavevector in observer frame: k_obs = (1, p)
	// Lorentz boost to world frame (boost velocity = +beta):
	//   k_world^0 = gamma * (1 + beta . p)
	//   k_world_spatial = p + beta*gamma + (gamma-1)/b2 * (beta.p) * beta
	bdotp := beta.Dot(p)

	kw0 := gamma * (1.0 + bdotp)

	factor := (gamma - 1.0) / b2 * bdotp
	kwSpatial := p.Add(beta.Scale(gamma)).Add(beta.Scale(factor))

	// Ray direction = negated photon propagation direction
	dir := kwSpatial.Neg().Normalize()
	doppler := 1.0 / kw0

	return AberrationResult{Dir: dir, Doppler: doppler}
}
