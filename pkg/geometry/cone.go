package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Cone is a capped cone along the local Y axis.
// Base at Y=0 with the given Radius, apex at Y=Height with radius 0.
// Use a Transformed wrapper to position and orient it.
type Cone struct {
	Radius float64
	Height float64
}

func (c *Cone) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	// Cone equation: x² + z² = (r/h)² * (h - y)²
	// Let k = r/h
	k := c.Radius / c.Height
	k2 := k * k

	// Substitute ray: p = o + t*d
	// (ox+t*dx)² + (oz+t*dz)² = k²*(h - oy - t*dy)²
	hy := c.Height - origin.Y
	a := dir.X*dir.X + dir.Z*dir.Z - k2*dir.Y*dir.Y
	b := 2*(origin.X*dir.X+origin.Z*dir.Z) + 2*k2*hy*dir.Y
	cc := origin.X*origin.X + origin.Z*origin.Z - k2*hy*hy

	var bestT float64
	var bestNormal vec.Vec3
	found := false

	disc := b*b - 4*a*cc
	if disc >= 0 && math.Abs(a) > 1e-12 {
		sqrtDisc := math.Sqrt(disc)
		for _, t := range [2]float64{(-b - sqrtDisc) / (2 * a), (-b + sqrtDisc) / (2 * a)} {
			if t < tMin || t > tMax {
				continue
			}
			p := origin.Add(dir.Scale(t))
			if p.Y >= 0 && p.Y <= c.Height {
				if !found || t < bestT {
					bestT = t
					// Cone normal: for point (x, y, z) on cone surface,
					// outward normal has radial component and downward Y component
					r := math.Sqrt(p.X*p.X + p.Z*p.Z)
					if r > 1e-12 {
						bestNormal = vec.Vec3{
							X: p.X / r,
							Y: k,
							Z: p.Z / r,
						}.Normalize()
					} else {
						bestNormal = vec.Vec3{Y: 1} // at apex
					}
					found = true
				}
				break
			}
		}
	}

	// Test base cap (Y=0)
	if math.Abs(dir.Y) > 1e-12 {
		t := -origin.Y / dir.Y
		if t >= tMin && t <= tMax {
			p := origin.Add(dir.Scale(t))
			if p.X*p.X+p.Z*p.Z <= c.Radius*c.Radius {
				if !found || t < bestT {
					bestT = t
					bestNormal = vec.Vec3{Y: -1}
					found = true
				}
			}
		}
	}

	if !found {
		return Hit{}, false
	}
	p := origin.Add(dir.Scale(bestT))
	h := Hit{T: bestT, Point: p}
	h.SetFaceNormal(dir, bestNormal)
	return h, true
}
