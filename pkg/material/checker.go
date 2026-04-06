package material

import (
	"math"
	"math/rand"

	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Checker is a Lambertian diffuse material with a checkerboard pattern
// alternating between two spectral reflectances. The pattern is computed
// in the tangent plane of the surface (perpendicular to the hit normal),
// so it produces a proper checkerboard on every flat face of any shape.
type Checker struct {
	Even, Odd spectrum.SPD
	Scale     float64 // size of each checker square in world units
}

func (c *Checker) Scatter(inDir vec.Vec3, hit geometry.Hit, rng *rand.Rand) ScatterResult {
	scattered := hit.Normal.Add(RandomUnitVec(rng))
	if scattered.LengthSq() < 1e-12 {
		scattered = hit.Normal
	}
	return ScatterResult{
		Scattered:   true,
		OutDir:      scattered.Normalize(),
		Reflectance: c.reflectanceAt(hit.Point, hit.Normal),
	}
}

func (c *Checker) Emitted(hit geometry.Hit) spectrum.SPD {
	return spectrum.SPD{}
}

func (c *Checker) reflectanceAt(p, n vec.Vec3) spectrum.SPD {
	// Build a tangent frame from the surface normal.
	ref := vec.Vec3{X: 1, Y: 0, Z: 0}
	if math.Abs(n.X) > 0.9 {
		ref = vec.Vec3{X: 0, Y: 1, Z: 0}
	}
	t1 := n.Cross(ref).Normalize()
	t2 := n.Cross(t1)

	inv := 1.0 / c.Scale
	iu := int(math.Floor(p.Dot(t1) * inv))
	iv := int(math.Floor(p.Dot(t2) * inv))
	if (iu+iv)%2 == 0 {
		return c.Even
	}
	return c.Odd
}
