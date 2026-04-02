package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Triangle is defined by three vertices. Uses the Moller-Trumbore algorithm.
type Triangle struct {
	V0, V1, V2 vec.Vec3
}

func (tri *Triangle) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	edge1 := tri.V1.Sub(tri.V0)
	edge2 := tri.V2.Sub(tri.V0)
	h := dir.Cross(edge2)
	a := edge1.Dot(h)

	if math.Abs(a) < 1e-12 {
		return Hit{}, false // ray parallel to triangle
	}

	f := 1.0 / a
	s := origin.Sub(tri.V0)
	u := f * s.Dot(h)
	if u < 0 || u > 1 {
		return Hit{}, false
	}

	q := s.Cross(edge1)
	v := f * dir.Dot(q)
	if v < 0 || u+v > 1 {
		return Hit{}, false
	}

	t := f * edge2.Dot(q)
	if t < tMin || t > tMax {
		return Hit{}, false
	}

	p := origin.Add(dir.Scale(t))
	normal := edge1.Cross(edge2).Normalize()
	hit := Hit{T: t, Point: p}
	hit.SetFaceNormal(dir, normal)
	return hit, true
}
