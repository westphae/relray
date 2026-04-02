package spectrum

import "math"

// Planck constants
const (
	planckH = 6.62607015e-34 // J·s
	planckC = 2.99792458e8   // m/s
	boltzK  = 1.380649e-23   // J/K
)

// Blackbody returns the spectral radiance of a blackbody at temperature T (Kelvin),
// normalized so that the Y component equals the given luminance.
// This uses the Planck function B(λ, T) = 2hc²/λ⁵ · 1/(exp(hc/λkT) - 1).
func Blackbody(tempK float64, luminance float64) SPD {
	var s SPD
	for i := range s {
		lambda := Wavelength(i) * 1e-9 // convert nm to meters
		s[i] = planckRadiance(lambda, tempK)
	}

	// Normalize to desired luminance (Y value)
	_, y, _ := s.ToXYZ()
	if y > 0 {
		s = s.Scale(luminance / y)
	}
	return s
}

// planckRadiance computes B(λ, T) in W·sr⁻¹·m⁻³.
func planckRadiance(lambdaM float64, tempK float64) float64 {
	a := 2 * planckH * planckC * planckC / math.Pow(lambdaM, 5)
	exponent := planckH * planckC / (lambdaM * boltzK * tempK)
	// Guard against overflow
	if exponent > 500 {
		return 0
	}
	return a / (math.Exp(exponent) - 1)
}
