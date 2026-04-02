package scene

import (
	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/material"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Object combines a shape with a material.
type Object struct {
	Shape    geometry.Shape
	Material material.Material
}

// Scene holds all objects and lights.
type Scene struct {
	Objects []Object
	Lights  []Light
	Sky     func(dir vec.Vec3) spectrum.SPD // optional sky/environment function
}

// Intersect finds the closest object hit by the ray.
func (s *Scene) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (geometry.Hit, material.Material, bool) {
	var closestHit geometry.Hit
	var closestMat material.Material
	found := false
	closest := tMax

	for i := range s.Objects {
		if h, ok := s.Objects[i].Shape.Intersect(origin, dir, tMin, closest); ok {
			closestHit = h
			closestMat = s.Objects[i].Material
			closest = h.T
			found = true
		}
	}
	return closestHit, closestMat, found
}
