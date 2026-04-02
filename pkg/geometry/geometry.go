package geometry

import "sif/gogs/eric/relray/pkg/vec"

// Hit records a ray-geometry intersection.
type Hit struct {
	T         float64  // ray parameter (distance along ray)
	Point     vec.Vec3 // world-space hit point
	Normal    vec.Vec3 // outward-facing unit surface normal
	FrontFace bool     // true if ray hit the front face
}

// SetFaceNormal sets the normal to always point against the incoming ray.
func (h *Hit) SetFaceNormal(rayDir, outwardNormal vec.Vec3) {
	h.FrontFace = rayDir.Dot(outwardNormal) < 0
	if h.FrontFace {
		h.Normal = outwardNormal
	} else {
		h.Normal = outwardNormal.Neg()
	}
}

// Shape is anything a ray can intersect.
type Shape interface {
	Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool)
}
