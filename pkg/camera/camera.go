package camera

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Camera defines the observer's position, orientation, and velocity.
type Camera struct {
	Position vec.Vec3
	LookAt   vec.Vec3
	Up       vec.Vec3
	VFOV     float64 // vertical field of view in degrees
	Aspect   float64 // width / height

	// Velocity in world frame as fraction of c.
	// |Beta| must be < 1.
	Beta vec.Vec3

	// Computed basis vectors (call Init before use)
	u, v, w          vec.Vec3
	halfW, halfH     float64
}

// Init computes the internal camera basis from the public fields.
// Must be called before RayDir.
func (c *Camera) Init() {
	theta := c.VFOV * math.Pi / 180.0
	c.halfH = math.Tan(theta / 2.0)
	c.halfW = c.halfH * c.Aspect

	c.w = c.Position.Sub(c.LookAt).Normalize() // points backward (away from look target)
	c.u = c.Up.Cross(c.w).Normalize()           // points right
	c.v = c.w.Cross(c.u)                         // points up
}

// RayDir returns the ray direction in the observer's rest frame for
// normalized screen coordinates (s, t) where (0,0) is bottom-left
// and (1,1) is top-right.
func (c *Camera) RayDir(s, t float64) vec.Vec3 {
	// Map (s,t) from [0,1] to [-halfW,halfW] x [-halfH,halfH]
	x := (2*s - 1) * c.halfW
	y := (2*t - 1) * c.halfH

	// Direction = x*u + y*v - w (toward the scene)
	dir := c.u.Scale(x).Add(c.v.Scale(y)).Sub(c.w)
	return dir.Normalize()
}
