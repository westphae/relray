package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Cylinder is a capped cylinder along the local Y axis.
// It extends from Y=0 to Y=Height, centered at X=0, Z=0.
// Use a Transformed wrapper to position and orient it in the scene.
type Cylinder struct {
	Radius float64
	Height float64
}

func (c *Cylinder) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	// Solve for intersection with infinite cylinder x² + z² = r²
	a := dir.X*dir.X + dir.Z*dir.Z
	b := 2 * (origin.X*dir.X + origin.Z*dir.Z)
	cc := origin.X*origin.X + origin.Z*origin.Z - c.Radius*c.Radius

	var bestT float64 = -1
	var bestNormal vec.Vec3
	found := false

	disc := b*b - 4*a*cc
	if disc >= 0 && a > 1e-12 {
		sqrtDisc := math.Sqrt(disc)
		for _, t := range [2]float64{(-b - sqrtDisc) / (2 * a), (-b + sqrtDisc) / (2 * a)} {
			if t < tMin || t > tMax {
				continue
			}
			y := origin.Y + t*dir.Y
			if y >= 0 && y <= c.Height {
				if !found || t < bestT {
					bestT = t
					p := origin.Add(dir.Scale(t))
					bestNormal = vec.Vec3{X: p.X, Z: p.Z}.Normalize()
					found = true
				}
				break // first valid t from sorted pair is closest
			}
		}
	}

	// Test caps (Y=0 and Y=Height)
	if math.Abs(dir.Y) > 1e-12 {
		for _, capY := range [2]float64{0, c.Height} {
			t := (capY - origin.Y) / dir.Y
			if t < tMin || t > tMax {
				continue
			}
			p := origin.Add(dir.Scale(t))
			if p.X*p.X+p.Z*p.Z <= c.Radius*c.Radius {
				if !found || t < bestT {
					bestT = t
					if capY == 0 {
						bestNormal = vec.Vec3{Y: -1}
					} else {
						bestNormal = vec.Vec3{Y: 1}
					}
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
