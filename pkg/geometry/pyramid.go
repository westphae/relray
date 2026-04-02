package geometry

import (
	"math"

	"sif/gogs/eric/relray/pkg/vec"
)

// Pyramid is a compound shape: a regular polygon base with triangular sides
// meeting at an apex. Defined in local space with base centered at origin
// on the XZ plane (Y=0), apex at (0, Height, 0).
//
// Use a Transformed wrapper to position and orient it.
type Pyramid struct {
	BaseRadius float64
	Height     float64
	Sides      int
	faces      []Triangle // computed on first use
}

func (p *Pyramid) ensureFaces() {
	if len(p.faces) > 0 {
		return
	}
	n := p.Sides
	if n < 3 {
		n = 3
	}
	apex := vec.Vec3{Y: p.Height}

	// Generate base vertices
	verts := make([]vec.Vec3, n)
	for i := range n {
		angle := 2 * math.Pi * float64(i) / float64(n)
		verts[i] = vec.Vec3{
			X: p.BaseRadius * math.Cos(angle),
			Z: p.BaseRadius * math.Sin(angle),
		}
	}

	// Side faces
	for i := range n {
		j := (i + 1) % n
		p.faces = append(p.faces, Triangle{V0: apex, V1: verts[i], V2: verts[j]})
	}

	// Base faces (fan triangulation)
	baseCenter := vec.Vec3{}
	for i := range n {
		j := (i + 1) % n
		// Wind opposite to sides so normal points down
		p.faces = append(p.faces, Triangle{V0: baseCenter, V1: verts[j], V2: verts[i]})
	}
}

func (p *Pyramid) Intersect(origin, dir vec.Vec3, tMin, tMax float64) (Hit, bool) {
	p.ensureFaces()

	var closest Hit
	found := false
	best := tMax

	for i := range p.faces {
		if h, ok := p.faces[i].Intersect(origin, dir, tMin, best); ok {
			closest = h
			best = h.T
			found = true
		}
	}
	return closest, found
}
