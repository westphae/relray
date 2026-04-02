package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Plane is an infinite plane defined by a point and a normal.
type Plane struct {
	Point  vec.Vec3
	Normal vec.Vec3 // must be unit length
}

func (p *Plane) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	denom := dir.Dot(p.Normal)
	if math.Abs(denom) < 1e-12 {
		return Hit{}, false // ray parallel to plane
	}
	t := p.Point.Sub(origin).Dot(p.Normal) / denom
	if t < tMin || t > tMax {
		return Hit{}, false
	}
	pt := origin.Add(dir.Scale(t))
	h := Hit{T: t, Point: pt}
	h.SetFaceNormal(dir, p.Normal)
	return h, true
}
