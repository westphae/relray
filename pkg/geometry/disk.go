package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Disk is a flat circular surface on the XZ plane (Y=0) with normal pointing +Y.
// Use a Transformed wrapper to position and orient it in the scene.
type Disk struct {
	Radius float64
}

func (d *Disk) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	if math.Abs(dir.Y) < 1e-12 {
		return Hit{}, false
	}
	t := -origin.Y / dir.Y
	if t < tMin || t > tMax {
		return Hit{}, false
	}
	p := origin.Add(dir.Scale(t))
	if p.X*p.X+p.Z*p.Z > d.Radius*d.Radius {
		return Hit{}, false
	}
	h := Hit{T: t, Point: p}
	h.SetFaceNormal(dir, vec.Vec3{Y: 1})
	return h, true
}
