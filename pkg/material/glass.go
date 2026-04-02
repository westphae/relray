package material

import (
	"math"
	"math/rand"

	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Glass is a dielectric material with refraction and Fresnel reflection.
// Uses Schlick's approximation for the Fresnel term.
type Glass struct {
	// IOR is the index of refraction (e.g., 1.5 for typical glass).
	IOR float64
	// Tint is the spectral transmittance of the glass (1.0 = perfectly clear).
	// Values < 1 at certain wavelengths create colored glass.
	Tint spectrum.SPD
}

func (g *Glass) Scatter(inDir vec.Vec3, hit geometry.Hit, rng *rand.Rand) ScatterResult {
	// Determine if we're entering or exiting the glass
	var etaRatio float64 // eta_incident / eta_transmitted
	if hit.FrontFace {
		etaRatio = 1.0 / g.IOR // air → glass
	} else {
		etaRatio = g.IOR // glass → air
	}

	unitDir := inDir.Normalize()
	cosI := math.Min(-unitDir.Dot(hit.Normal), 1.0)
	sin2T := etaRatio * etaRatio * (1.0 - cosI*cosI)

	// Total internal reflection check
	if sin2T > 1.0 {
		reflected := unitDir.Reflect(hit.Normal)
		return ScatterResult{
			Scattered:   true,
			OutDir:      reflected.Normalize(),
			Reflectance: g.Tint,
		}
	}

	// Schlick's approximation for Fresnel reflectance
	reflectance := schlick(cosI, etaRatio)

	// Probabilistically choose reflection vs refraction
	if rng.Float64() < reflectance {
		reflected := unitDir.Reflect(hit.Normal)
		return ScatterResult{
			Scattered:   true,
			OutDir:      reflected.Normalize(),
			Reflectance: g.Tint,
		}
	}

	// Refract
	refracted := refract(unitDir, hit.Normal, etaRatio)
	return ScatterResult{
		Scattered:   true,
		OutDir:      refracted.Normalize(),
		Reflectance: g.Tint,
	}
}

func (g *Glass) Emitted(hit geometry.Hit) spectrum.SPD {
	return spectrum.SPD{}
}

// schlick computes Schlick's approximation to the Fresnel reflectance.
func schlick(cosine float64, etaRatio float64) float64 {
	r0 := (1 - etaRatio) / (1 + etaRatio)
	r0 = r0 * r0
	return r0 + (1-r0)*math.Pow(1-cosine, 5)
}

// refract computes the refracted direction using Snell's law.
func refract(uv, n vec.Vec3, etaRatio float64) vec.Vec3 {
	cosTheta := math.Min(-uv.Dot(n), 1.0)
	rPerp := uv.Add(n.Scale(cosTheta)).Scale(etaRatio)
	rParallel := n.Scale(-math.Sqrt(math.Abs(1.0 - rPerp.LengthSq())))
	return rPerp.Add(rParallel)
}
