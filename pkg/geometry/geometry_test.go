package geometry

import (
	"math"
	"testing"

	"sif/gogs/eric/relray/pkg/vec"
)

const eps = 1e-9

func TestSphereIntersect(t *testing.T) {
	// Sphere at origin, radius 1. Ray from (0,0,-5) along +Z should hit at Z=-1.
	s := &Sphere{Radius: 1}
	h, ok := s.Intersect(vec.Vec3{Z: -5}, vec.Vec3{Z: 1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit")
	}
	if math.Abs(h.T-4.0) > eps {
		t.Errorf("T = %v, want 4", h.T)
	}
	if !h.FrontFace {
		t.Error("expected front face")
	}
}

func TestSphereMiss(t *testing.T) {
	s := &Sphere{Radius: 1}
	_, ok := s.Intersect(vec.Vec3{X: 5}, vec.Vec3{Z: 1}, 0.001, 1e9)
	if ok {
		t.Error("expected miss")
	}
}

func TestSphereInside(t *testing.T) {
	s := &Sphere{Radius: 10}
	h, ok := s.Intersect(vec.Vec3{}, vec.Vec3{Z: 1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit from inside")
	}
	if h.FrontFace {
		t.Error("expected back face when inside sphere")
	}
}

func TestPlaneIntersect(t *testing.T) {
	// Default plane at Y=0. Ray from (0,1,0) going down should hit at T=1.
	p := &Plane{}
	h, ok := p.Intersect(vec.Vec3{Y: 1}, vec.Vec3{Y: -1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit")
	}
	if math.Abs(h.T-1.0) > eps {
		t.Errorf("T = %v, want 1", h.T)
	}
	if math.Abs(h.Point.Y) > eps {
		t.Errorf("hit Y = %v, want 0", h.Point.Y)
	}
}

func TestPlaneParallel(t *testing.T) {
	p := &Plane{}
	_, ok := p.Intersect(vec.Vec3{Y: 1}, vec.Vec3{X: 1}, 0.001, 1e9)
	if ok {
		t.Error("expected miss for parallel ray")
	}
}

func TestBoxIntersect(t *testing.T) {
	b := &Box{Size: vec.Vec3{X: 2, Y: 2, Z: 2}} // extends -1 to +1
	h, ok := b.Intersect(vec.Vec3{Z: -5}, vec.Vec3{Z: 1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit")
	}
	if math.Abs(h.T-4.0) > eps {
		t.Errorf("T = %v, want 4", h.T)
	}
}

func TestTransformedSphere(t *testing.T) {
	// Sphere at origin, translated to (0,0,5)
	s := NewTransformed(&Sphere{Radius: 1}, vec.Vec3{Z: 5}, vec.Identity())
	h, ok := s.Intersect(vec.Vec3{}, vec.Vec3{Z: 1}, 0.001, 1e9)
	if !ok {
		t.Fatal("expected hit")
	}
	if math.Abs(h.T-4.0) > eps {
		t.Errorf("T = %v, want 4", h.T)
	}
}
