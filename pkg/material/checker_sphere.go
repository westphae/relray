package material

import (
	"math"
	"math/rand"

	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// CheckerSphere is a Lambertian diffuse material with a checkerboard pattern
// mapped onto a sphere using latitude/longitude coordinates derived from the
// surface normal. NumSquares controls how many checker divisions there are
// around the equator (and half that many pole-to-pole).
type CheckerSphere struct {
	Even, Odd  spectrum.SPD
	NumSquares int // divisions around equator (default 8)
}

func (c *CheckerSphere) Scatter(inDir vec.Vec3, hit geometry.Hit, rng *rand.Rand) ScatterResult {
	scattered := hit.Normal.Add(RandomUnitVec(rng))
	if scattered.LengthSq() < 1e-12 {
		scattered = hit.Normal
	}
	return ScatterResult{
		Scattered:   true,
		OutDir:      scattered.Normalize(),
		Reflectance: c.reflectanceAt(hit.Normal),
	}
}

func (c *CheckerSphere) Emitted(hit geometry.Hit) spectrum.SPD {
	return spectrum.SPD{}
}

func (c *CheckerSphere) reflectanceAt(normal vec.Vec3) spectrum.SPD {
	n := c.NumSquares
	if n <= 0 {
		n = 8
	}

	// Latitude: -π/2 to π/2, longitude: -π to π
	lat := math.Asin(clamp(normal.Y, -1, 1))
	lon := math.Atan2(normal.Z, normal.X)

	// Map to grid squares
	latDiv := int(math.Floor(lat * float64(n) / math.Pi))
	lonDiv := int(math.Floor(lon * float64(n) / math.Pi))

	if (latDiv+lonDiv)%2 == 0 {
		return c.Even
	}
	return c.Odd
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
