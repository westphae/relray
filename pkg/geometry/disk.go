package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Disk is a flat circular surface defined by center, normal, and radius.
type Disk struct {
	Center vec.Vec3
	Normal vec.Vec3 // must be unit length
	Radius float64
}

func (d *Disk) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	denom := dir.Dot(d.Normal)
	if math.Abs(denom) < 1e-12 {
		return Hit{}, false
	}
	t := d.Center.Sub(origin).Dot(d.Normal) / denom
	if t < tMin || t > tMax {
		return Hit{}, false
	}
	p := origin.Add(dir.Scale(t))
	if p.Sub(d.Center).LengthSq() > d.Radius*d.Radius {
		return Hit{}, false
	}
	h := Hit{T: t, Point: p}
	h.SetFaceNormal(dir, d.Normal)
	return h, true
}
