package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Sphere is a geometric sphere defined by center and radius.
type Sphere struct {
	Center vec.Vec3
	Radius float64
}

func (s *Sphere) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	oc := origin.Sub(s.Center)
	a := dir.LengthSq()
	halfB := oc.Dot(dir)
	c := oc.LengthSq() - s.Radius*s.Radius
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
	outward := p.Sub(s.Center).Scale(1.0 / s.Radius)
	h := Hit{T: t, Point: p}
	h.SetFaceNormal(dir, outward)
	return h, true
}
