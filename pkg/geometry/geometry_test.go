package geometry

import (
	"math"
	"testing"

	"sif/gogs/eric/relray/pkg/vec"
)

const eps = 1e-9

func TestSphereIntersect(t *testing.T) {
	s := &Sphere{Center: vec.Vec3{Z: 5}, Radius: 1}

	// Ray along +Z should hit sphere at Z=4
	h, ok := s.Intersect(vec.Vec3{}, vec.Vec3{Z: 1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit")
	}
	if math.Abs(h.T-4.0) > eps {
		t.Errorf("T = %v, want 4", h.T)
	}
	if !h.FrontFace {
		t.Error("expected front face")
	}
	// Normal should point toward origin (back toward ray)
	if h.Normal.Z >= 0 {
		t.Errorf("normal.Z = %v, expected negative", h.Normal.Z)
	}
}

func TestSphereMiss(t *testing.T) {
	s := &Sphere{Center: vec.Vec3{Z: 5}, Radius: 1}
	_, ok := s.Intersect(vec.Vec3{}, vec.Vec3{X: 1}, 0.001, 1e9)
	if ok {
		t.Error("expected miss")
	}
}

func TestSphereInside(t *testing.T) {
	s := &Sphere{Center: vec.Vec3{}, Radius: 10}
	h, ok := s.Intersect(vec.Vec3{}, vec.Vec3{Z: 1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit from inside")
	}
	if h.FrontFace {
		t.Error("expected back face when inside sphere")
	}
}

func TestPlaneIntersect(t *testing.T) {
	p := &Plane{Point: vec.Vec3{Y: -1}, Normal: vec.Vec3{Y: 1}}

	// Ray going down should hit floor
	h, ok := p.Intersect(vec.Vec3{}, vec.Vec3{Y: -1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit")
	}
	if math.Abs(h.T-1.0) > eps {
		t.Errorf("T = %v, want 1", h.T)
	}
	if math.Abs(h.Point.Y-(-1.0)) > eps {
		t.Errorf("hit Y = %v, want -1", h.Point.Y)
	}
}

func TestPlaneParallel(t *testing.T) {
	p := &Plane{Point: vec.Vec3{Y: -1}, Normal: vec.Vec3{Y: 1}}
	_, ok := p.Intersect(vec.Vec3{}, vec.Vec3{X: 1}, 0.001, 1e9)
	if ok {
		t.Error("expected miss for parallel ray")
	}
}
