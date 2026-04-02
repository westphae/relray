package geometry

import "sif/gogs/eric/relray/pkg/vec"

// Transformed wraps a Shape with a rotation and translation.
// The wrapped shape is defined in local space; the transform maps
// from local space to world space.
type Transformed struct {
	Shape    Shape
	Position vec.Vec3 // translation (local origin in world space)
	Rotation vec.Mat3 // rotation matrix (local → world)
	InvRot   vec.Mat3 // inverse rotation (world → local) = Rotation.Transpose()
}

// NewTransformed creates a Transformed shape. Precomputes the inverse rotation.
func NewTransformed(shape Shape, position vec.Vec3, rotation vec.Mat3) *Transformed {
	return &Transformed{
		Shape:    shape,
		Position: position,
		Rotation: rotation,
		InvRot:   rotation.Transpose(),
	}
}

func (t *Transformed) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	// Transform ray from world space to local space
	localOrigin := t.InvRot.MulVec(origin.Sub(t.Position))
	localDir := t.InvRot.MulVec(dir)

	h, ok := t.Shape.Intersect(localOrigin, localDir, tMin, tMax)
	if !ok {
		return Hit{}, false
	}

	// Transform hit back to world space
	h.Point = t.Rotation.MulVec(h.Point).Add(t.Position)
	h.Normal = t.Rotation.MulVec(h.Normal).Normalize()
	return h, true
}
