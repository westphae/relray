package spectrum

import "math"

const (
	LambdaMin  = 380.0 // nm
	LambdaMax  = 780.0 // nm
	LambdaStep = 5.0   // nm
	NumBands   = 81    // (780-380)/5 + 1
)

// SPD represents a spectral power distribution sampled at 5nm intervals from 380nm to 780nm.
type SPD [NumBands]float64

// Wavelength returns the wavelength in nm for band index i.
func Wavelength(i int) float64 {
	return LambdaMin + float64(i)*LambdaStep
}

// BandIndex returns the fractional band index for a wavelength.
func BandIndex(lambda float64) float64 {
	return (lambda - LambdaMin) / LambdaStep
}

func (s SPD) Add(other SPD) SPD {
	var r SPD
	for i := range r {
		r[i] = s[i] + other[i]
	}
	return r
}

func (s SPD) Mul(other SPD) SPD {
	var r SPD
	for i := range r {
		r[i] = s[i] * other[i]
	}
	return r
}

func (s SPD) Scale(f float64) SPD {
	var r SPD
	for i := range r {
		r[i] = s[i] * f
	}
	return r
}

// Shift returns a new SPD with wavelengths scaled by factor.
// factor > 1 = redshift (wavelengths increase), factor < 1 = blueshift.
// Uses linear interpolation for resampling. Energy that shifts outside
// the visible range is lost, which is physically correct.
func (s SPD) Shift(factor float64) SPD {
	if factor == 1.0 {
		return s
	}
	var r SPD
	for i := range r {
		// What original wavelength maps to this band after shifting?
		// new_lambda = old_lambda * factor, so old_lambda = new_lambda / factor
		origLambda := Wavelength(i) / factor
		idx := BandIndex(origLambda)
		if idx < 0 || idx >= float64(NumBands-1) {
			continue // outside visible range
		}
		lo := int(idx)
		frac := idx - float64(lo)
		r[i] = s[lo]*(1-frac) + s[lo+1]*frac
	}
	return r
}

// ToXYZ integrates the SPD against the CIE 1931 2° color matching functions.
func (s SPD) ToXYZ() (x, y, z float64) {
	for i := 0; i < NumBands; i++ {
		x += s[i] * cieX[i]
		y += s[i] * cieY[i]
		z += s[i] * cieZ[i]
	}
	x *= LambdaStep
	y *= LambdaStep
	z *= LambdaStep
	return
}

// XYZToLinearRGB converts CIE XYZ to linear sRGB (not gamma-corrected).
func XYZToLinearRGB(x, y, z float64) (r, g, b float64) {
	// sRGB D65 matrix (IEC 61966-2-1)
	r = 3.2406*x - 1.5372*y - 0.4986*z
	g = -0.9689*x + 1.8758*y + 0.0415*z
	b = 0.0557*x - 0.2040*y + 1.0570*z
	return
}

// sRGBGamma applies the sRGB gamma curve to a linear channel value.
func sRGBGamma(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

// XYZToSRGB converts CIE XYZ to 8-bit sRGB with gamma correction and clamping.
func XYZToSRGB(x, y, z float64) (r, g, b uint8) {
	lr, lg, lb := XYZToLinearRGB(x, y, z)
	return toUint8(sRGBGamma(lr)), toUint8(sRGBGamma(lg)), toUint8(sRGBGamma(lb))
}

func toUint8(v float64) uint8 {
	c := v * 255.0
	if c < 0 {
		return 0
	}
	if c > 255 {
		return 255
	}
	return uint8(c + 0.5)
}

// D65 returns the CIE standard illuminant D65 spectral power distribution,
// normalized so that Y=1 when integrated against the CIE Y color matching function.
func D65() SPD {
	// Normalize so that integrating d65Raw against cieY gives Y=1
	var sum float64
	for i := 0; i < NumBands; i++ {
		sum += d65Raw[i] * cieY[i]
	}
	sum *= LambdaStep
	var s SPD
	for i := range s {
		s[i] = d65Raw[i] / sum
	}
	return s
}

// Constant returns an SPD with all bands set to v.
func Constant(v float64) SPD {
	var s SPD
	for i := range s {
		s[i] = v
	}
	return s
}

// Monochromatic returns an SPD with energy concentrated at the given wavelength.
// The energy is distributed across the two nearest bands via linear interpolation.
func Monochromatic(lambda float64, power float64) SPD {
	var s SPD
	idx := BandIndex(lambda)
	if idx < 0 || idx >= float64(NumBands-1) {
		return s
	}
	lo := int(idx)
	frac := idx - float64(lo)
	s[lo] = power * (1 - frac)
	if lo+1 < NumBands {
		s[lo+1] = power * frac
	}
	return s
}

// FromRGB creates an SPD from linear sRGB values using a CIE basis function approach.
// Converts RGB→XYZ, then finds SPD = a·x̄ + b·ȳ + c·z̄ that produces the target XYZ,
// with negative values clamped to zero.
func FromRGB(r, g, b float64) SPD {
	// sRGB to XYZ (D65 reference white)
	tx := 0.4124564*r + 0.3575761*g + 0.1804375*b
	ty := 0.2126729*r + 0.7151522*g + 0.0721750*b
	tz := 0.0193339*r + 0.1191920*g + 0.9503041*b

	// Solve for coefficients [a,b,c] such that:
	//   ∫(a·x̄ + b·ȳ + c·z̄)·x̄ dλ = tx  (and similarly for ty, tz)
	// This requires M·[a,b,c]^T = [tx,ty,tz]^T where M is the Gram matrix.
	a, bb2, cc := solveGram(tx, ty, tz)

	var s SPD
	for i := range s {
		v := a*cieX[i] + bb2*cieY[i] + cc*cieZ[i]
		if v < 0 {
			v = 0 // clamp negatives for physical plausibility
		}
		s[i] = v
	}
	return s
}

// gramInv is the precomputed inverse of the Gram matrix M_ij = Σ f_i[k]*f_j[k] * step
// where f_0=x̄, f_1=ȳ, f_2=z̄. Computed once at init.
var gramInv [3][3]float64

func init() {
	// Compute Gram matrix
	var m [3][3]float64
	cmf := [3]*[NumBands]float64{&cieX, &cieY, &cieZ}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			var sum float64
			for k := 0; k < NumBands; k++ {
				sum += cmf[i][k] * cmf[j][k]
			}
			m[i][j] = sum * LambdaStep
		}
	}
	// Invert 3x3 matrix
	gramInv = invert3x3(m)
}

func solveGram(tx, ty, tz float64) (a, b, c float64) {
	a = gramInv[0][0]*tx + gramInv[0][1]*ty + gramInv[0][2]*tz
	b = gramInv[1][0]*tx + gramInv[1][1]*ty + gramInv[1][2]*tz
	c = gramInv[2][0]*tx + gramInv[2][1]*ty + gramInv[2][2]*tz
	return
}

func invert3x3(m [3][3]float64) [3][3]float64 {
	a, b, c := m[0][0], m[0][1], m[0][2]
	d, e, f := m[1][0], m[1][1], m[1][2]
	g, h, k := m[2][0], m[2][1], m[2][2]

	det := a*(e*k-f*h) - b*(d*k-f*g) + c*(d*h-e*g)

	var inv [3][3]float64
	inv[0][0] = (e*k - f*h) / det
	inv[0][1] = (c*h - b*k) / det
	inv[0][2] = (b*f - c*e) / det
	inv[1][0] = (f*g - d*k) / det
	inv[1][1] = (a*k - c*g) / det
	inv[1][2] = (c*d - a*f) / det
	inv[2][0] = (d*h - e*g) / det
	inv[2][1] = (b*g - a*h) / det
	inv[2][2] = (a*e - b*d) / det
	return inv
}
