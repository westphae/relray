package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Box is an axis-aligned bounding box defined by min and max corners.
type Box struct {
	Min, Max vec.Vec3
}

func (b *Box) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	// Slab method
	invD := vec.Vec3{
		X: 1.0 / dir.X,
		Y: 1.0 / dir.Y,
		Z: 1.0 / dir.Z,
	}

	t0x := (b.Min.X - origin.X) * invD.X
	t1x := (b.Max.X - origin.X) * invD.X
	if invD.X < 0 {
		t0x, t1x = t1x, t0x
	}

	t0y := (b.Min.Y - origin.Y) * invD.Y
	t1y := (b.Max.Y - origin.Y) * invD.Y
	if invD.Y < 0 {
		t0y, t1y = t1y, t0y
	}

	t0z := (b.Min.Z - origin.Z) * invD.Z
	t1z := (b.Max.Z - origin.Z) * invD.Z
	if invD.Z < 0 {
		t0z, t1z = t1z, t0z
	}

	tNear := math.Max(t0x, math.Max(t0y, t0z))
	tFar := math.Min(t1x, math.Min(t1y, t1z))

	if tNear > tFar || tFar < tMin || tNear > tMax {
		return Hit{}, false
	}

	t := tNear
	if t < tMin {
		t = tFar
		if t > tMax {
			return Hit{}, false
		}
	}

	p := origin.Add(dir.Scale(t))

	// Determine which face was hit by finding which component of the hit point
	// is closest to a face
	var normal vec.Vec3
	const bias = 1e-6
	switch {
	case math.Abs(p.X-b.Min.X) < bias:
		normal = vec.Vec3{X: -1}
	case math.Abs(p.X-b.Max.X) < bias:
		normal = vec.Vec3{X: 1}
	case math.Abs(p.Y-b.Min.Y) < bias:
		normal = vec.Vec3{Y: -1}
	case math.Abs(p.Y-b.Max.Y) < bias:
		normal = vec.Vec3{Y: 1}
	case math.Abs(p.Z-b.Min.Z) < bias:
		normal = vec.Vec3{Z: -1}
	default:
		normal = vec.Vec3{Z: 1}
	}

	h := Hit{T: t, Point: p}
	h.SetFaceNormal(dir, normal)
	return h, true
}
