package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Box is an axis-aligned box centered at the origin with the given size.
// It extends from -Size/2 to +Size/2 on each axis.
// Use a Transformed wrapper to position and orient it in the scene.
type Box struct {
	Size vec.Vec3
}

func (b *Box) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	halfX := b.Size.X / 2
	halfY := b.Size.Y / 2
	halfZ := b.Size.Z / 2

	invD := vec.Vec3{
		X: 1.0 / dir.X,
		Y: 1.0 / dir.Y,
		Z: 1.0 / dir.Z,
	}

	t0x := (-halfX - origin.X) * invD.X
	t1x := (halfX - origin.X) * invD.X
	if invD.X < 0 {
		t0x, t1x = t1x, t0x
	}

	t0y := (-halfY - origin.Y) * invD.Y
	t1y := (halfY - origin.Y) * invD.Y
	if invD.Y < 0 {
		t0y, t1y = t1y, t0y
	}

	t0z := (-halfZ - origin.Z) * invD.Z
	t1z := (halfZ - origin.Z) * invD.Z
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

	var normal vec.Vec3
	const bias = 1e-6
	switch {
	case math.Abs(p.X+halfX) < bias:
		normal = vec.Vec3{X: -1}
	case math.Abs(p.X-halfX) < bias:
		normal = vec.Vec3{X: 1}
	case math.Abs(p.Y+halfY) < bias:
		normal = vec.Vec3{Y: -1}
	case math.Abs(p.Y-halfY) < bias:
		normal = vec.Vec3{Y: 1}
	case math.Abs(p.Z+halfZ) < bias:
		normal = vec.Vec3{Z: -1}
	default:
		normal = vec.Vec3{Z: 1}
	}

	h := Hit{T: t, Point: p}
	h.SetFaceNormal(dir, normal)
	return h, true
}
