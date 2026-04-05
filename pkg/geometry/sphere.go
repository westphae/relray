package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Sphere is centered at the origin with the given radius.
// Use a Transformed wrapper to position it in the scene.
type Sphere struct {
	Radius float64
}

func (s *Sphere) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	a := dir.LengthSq()
	halfB := origin.Dot(dir)
	c := origin.LengthSq() - s.Radius*s.Radius
	disc := halfB*halfB - a*c
	if disc < 0 {
		return Hit{}, false
	}

	sqrtDisc := math.Sqrt(disc)
	t := (-halfB - sqrtDisc) / a
	if t < tMin || t > tMax {
		t = (-halfB + sqrtDisc) / a
		if t < tMin || t > tMax {
			return Hit{}, false
		}
	}

	p := origin.Add(dir.Scale(t))
	outward := p.Scale(1.0 / s.Radius)
	h := Hit{T: t, Point: p}
	h.SetFaceNormal(dir, outward)
	return h, true
}
