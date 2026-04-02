package scene

import (
	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/material"
	"sif/gogs/eric/relray/pkg/retarded"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Object combines a shape with a material.
type Object struct {
	Shape    geometry.Shape
	Material material.Material
}

// MovingObject is an object with a time-dependent position.
// The Shape is defined in the object's local frame (centered at origin).
// The Trajectory gives the object's center position as a function of time.
type MovingObject struct {
	Shape      geometry.Shape
	Material   material.Material
	Trajectory retarded.Trajectory
}

// Scene holds all objects and lights.
type Scene struct {
	Name          string
	Objects       []Object
	MovingObjects []MovingObject
	Lights        []Light
	Time          float64                       // current scene time (for moving objects)
	Sky           func(dir vec.Vec3) spectrum.SPD // optional sky/environment function
}

// Intersect finds the closest object hit by the ray, including moving objects
// at their retarded-time positions.
func (s *Scene) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (geometry.Hit, material.Material, bool) {
	var closestHit geometry.Hit
	var closestMat material.Material
	found := false
	closest := tMax

	// Static objects
	for i := range s.Objects {
		if h, ok := s.Objects[i].Shape.Intersect(origin, dir, tMin, closest); ok {
			closestHit = h
			closestMat = s.Objects[i].Material
			closest = h.T
			found = true
		}
	}

	// Moving objects: solve retarded time, then intersect with object at retarded position.
	// For each moving object, we approximate by first solving retarded time for a point
	// along the ray (the ray-parameter midpoint of the current search range), then
	// intersecting with the shape translated to that retarded position.
	for i := range s.MovingObjects {
		mo := &s.MovingObjects[i]

		// Estimate where along the ray we might hit this object.
		// Use the object's current position as a starting point for the retarded-time solve.
		// The observer "position" for the retarded-time calculation is where the photon
		// needs to arrive — for a ray, this is the ray origin (the camera or last bounce point).
		tEmit, objPos, ok := retarded.Solve(origin, s.Time, mo.Trajectory)
		if !ok {
			continue
		}

		// Intersect ray with shape translated to retarded position
		localOrigin := origin.Sub(objPos)
		if h, ok := mo.Shape.Intersect(localOrigin, dir, tMin, closest); ok {
			h.Point = h.Point.Add(objPos)
			// Compute source velocity at retarded time (as fraction of c)
			h.SourceVelocity = retarded.Velocity(mo.Trajectory, tEmit)
			closestHit = h
			closestMat = mo.Material
			closest = h.T
			found = true
		}
	}

	return closestHit, closestMat, found
}
