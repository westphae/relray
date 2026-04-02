package material

import (
	"math"

	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Checker is a Lambertian diffuse material with a checkerboard pattern
// alternating between two spectral reflectances. The pattern is defined
// in the XZ plane (horizontal), with configurable scale.
type Checker struct {
	Even, Odd spectrum.SPD
	Scale     float64 // size of each checker square in world units
}

func (c *Checker) Scatter(inDir vec.Vec3, hit geometry.Hit) ScatterResult {
	scattered := hit.Normal.Add(randomUnitVec())
	if scattered.LengthSq() < 1e-12 {
		scattered = hit.Normal
	}
	return ScatterResult{
		Scattered:   true,
		OutDir:      scattered.Normalize(),
		Reflectance: c.reflectanceAt(hit.Point),
	}
}

func (c *Checker) Emitted(hit geometry.Hit) spectrum.SPD {
	return spectrum.SPD{}
}

func (c *Checker) reflectanceAt(p vec.Vec3) spectrum.SPD {
	inv := 1.0 / c.Scale
	ix := int(math.Floor(p.X * inv))
	iz := int(math.Floor(p.Z * inv))
	if (ix+iz)%2 == 0 {
		return c.Even
	}
	return c.Odd
}
