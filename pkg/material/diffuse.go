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

func (d *Diffuse) Scatter(inDir vec.Vec3, hit geometry.Hit, rng *rand.Rand) ScatterResult {
	scattered := hit.Normal.Add(RandomUnitVec(rng))
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

// RandomUnitVec returns a uniformly distributed random unit vector.
func RandomUnitVec(rng *rand.Rand) vec.Vec3 {
	for {
		v := vec.Vec3{
			X: 2*rng.Float64() - 1,
			Y: 2*rng.Float64() - 1,
			Z: 2*rng.Float64() - 1,
		}
		l2 := v.LengthSq()
		if l2 > 1e-6 && l2 <= 1 {
			return v.Scale(1.0 / math.Sqrt(l2))
		}
	}
}
