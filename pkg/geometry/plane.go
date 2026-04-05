package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Plane is an infinite plane on the XZ plane (Y=0) with normal pointing +Y.
// Use a Transformed wrapper to position and orient it in the scene.
type Plane struct{}

func (p *Plane) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	if math.Abs(dir.Y) < 1e-12 {
		return Hit{}, false // ray parallel to plane
	}
	t := -origin.Y / dir.Y
	if t < tMin || t > tMax {
		return Hit{}, false
	}
	pt := origin.Add(dir.Scale(t))
	h := Hit{T: t, Point: pt}
	h.SetFaceNormal(dir, vec.Vec3{Y: 1})
	return h, true
}
