package material

import (
	"math/rand"

	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Mirror is a perfect specular reflector with spectral reflectance.
type Mirror struct {
	Reflectance spectrum.SPD
}

func (m *Mirror) Scatter(inDir vec.Vec3, hit geometry.Hit, rng *rand.Rand) ScatterResult {
	reflected := inDir.Reflect(hit.Normal)
	return ScatterResult{
		Scattered:   true,
		OutDir:      reflected.Normalize(),
		Reflectance: m.Reflectance,
	}
}

func (m *Mirror) Emitted(hit geometry.Hit) spectrum.SPD {
	return spectrum.SPD{}
}
