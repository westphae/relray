package material

import (
	"math"
	"math/rand"

	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Diffuse is a Lambertian diffuse material with spectral reflectance.
type Diffuse struct {
	Reflectance spectrum.SPD
}

func (d *Diffuse) Scatter(inDir vec.Vec3, hit geometry.Hit) ScatterResult {
	// Cosine-weighted hemisphere sampling via normal + random unit vector
	scattered := hit.Normal.Add(randomUnitVec())
	// Catch degenerate case where random vector cancels normal
	if scattered.LengthSq() < 1e-12 {
		scattered = hit.Normal
	}
	return ScatterResult{
		Scattered:   true,
		OutDir:      scattered.Normalize(),
		Reflectance: d.Reflectance,
	}
}

func (d *Diffuse) Emitted(hit geometry.Hit) spectrum.SPD {
	return spectrum.SPD{}
}

func randomUnitVec() vec.Vec3 {
	for {
		v := vec.Vec3{
			X: 2*rand.Float64() - 1,
			Y: 2*rand.Float64() - 1,
			Z: 2*rand.Float64() - 1,
		}
		l2 := v.LengthSq()
		if l2 > 1e-6 && l2 <= 1 {
			return v.Scale(1.0 / math.Sqrt(l2))
		}
	}
}
