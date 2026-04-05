package spectrum

import (
	"math"

	"github.com/viterin/vek"
)

const (
	LambdaMin  = 200.0  // nm
	LambdaMax  = 2000.0 // nm
	LambdaStep = 5.0    // nm
	NumBands   = 361    // (2000-200)/5 + 1

	// Visible sub-range (for CIE integration — CIE functions are zero outside this)
	visibleMin  = 380.0
	visibleMax  = 780.0
	visibleStart = 36  // band index of 380nm: (380-200)/5
	visibleEnd   = 116 // band index of 780nm: (780-200)/5
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
	vek.Add_Into(r[:], s[:], other[:])
	return r
}

func (s SPD) Mul(other SPD) SPD {
	var r SPD
	vek.Mul_Into(r[:], s[:], other[:])
	return r
}

func (s SPD) Scale(f float64) SPD {
	var r SPD
	vek.MulNumber_Into(r[:], s[:], f)
	return r
}

// In-place variants for hot paths — avoid copying 2888-byte arrays.

func (s *SPD) AddInPlace(other *SPD) {
	vek.Add_Inplace(s[:], other[:])
}

func (s *SPD) MulInPlace(other *SPD) {
	vek.Mul_Inplace(s[:], other[:])
}

func (s *SPD) ScaleInPlace(f float64) {
	vek.MulNumber_Inplace(s[:], f)
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
	// Precompute: for output band i at wavelength LambdaMin + i*LambdaStep,
	// the source wavelength is (LambdaMin + i*LambdaStep) / factor.
	// The source band index is ((LambdaMin + i*LambdaStep)/factor - LambdaMin) / LambdaStep
	//   = (LambdaMin/factor - LambdaMin)/LambdaStep + i/factor
	//   = startIdx + i * step
	invFactor := 1.0 / factor
	startIdx := (LambdaMin*invFactor - LambdaMin) / LambdaStep
	step := invFactor
	maxIdx := float64(NumBands - 1)

	idx := startIdx
	for i := range r {
		if idx >= 0 && idx < maxIdx {
			lo := int(idx)
			frac := idx - float64(lo)
			r[i] = math.FMA(s[lo+1]-s[lo], frac, s[lo])
		}
		idx += step
	}
	return r
}

// ToXYZ integrates the SPD against the CIE 1931 2° color matching functions.
// Only integrates over the visible range (380-780nm) since CIE values are zero outside.
func (s SPD) ToXYZ() (x, y, z float64) {
	vis := s[visibleStart : visibleEnd+1]
	x = vek.Dot(vis, cieX[visibleStart:visibleEnd+1]) * LambdaStep
	y = vek.Dot(vis, cieY[visibleStart:visibleEnd+1]) * LambdaStep
	z = vek.Dot(vis, cieZ[visibleStart:visibleEnd+1]) * LambdaStep
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

// FromRGB creates a reflectance SPD from linear sRGB values.
//
// Uses three Gaussian basis reflectances calibrated so that under D65 illumination,
// the round-trip through CIE XYZ → sRGB reproduces the input colors correctly.
func FromRGB(r, g, b float64) SPD {
	// Target XYZ under D65 illumination
	tx := 0.4124564*r + 0.3575761*g + 0.1804375*b
	ty := 0.2126729*r + 0.7151522*g + 0.0721750*b
	tz := 0.0193339*r + 0.1191920*g + 0.9503041*b

	// Solve for coefficients: c = basisXYZInv * [tx, ty, tz]
	cr := basisXYZInv[0][0]*tx + basisXYZInv[0][1]*ty + basisXYZInv[0][2]*tz
	cg := basisXYZInv[1][0]*tx + basisXYZInv[1][1]*ty + basisXYZInv[1][2]*tz
	cb := basisXYZInv[2][0]*tx + basisXYZInv[2][1]*ty + basisXYZInv[2][2]*tz

	var s SPD
	for i := range s {
		v := cr*basisR[i] + cg*basisG[i] + cb*basisB[i]
		if v < 0 {
			v = 0
		}
		s[i] = v
	}
	return s
}

// FromReflectanceCurve creates an SPD from a set of (wavelength_nm, value) pairs.
// The pairs are linearly interpolated to fill all bands. Wavelengths outside the
// given range clamp to the nearest endpoint value. Points must be sorted by wavelength.
func FromReflectanceCurve(points [][2]float64) SPD {
	if len(points) == 0 {
		return SPD{}
	}
	var s SPD
	for i := range s {
		lambda := Wavelength(i)
		s[i] = interpolatePoints(points, lambda)
	}
	return s
}

func interpolatePoints(points [][2]float64, lambda float64) float64 {
	if lambda <= points[0][0] {
		return points[0][1]
	}
	if lambda >= points[len(points)-1][0] {
		return points[len(points)-1][1]
	}
	for j := 1; j < len(points); j++ {
		if lambda <= points[j][0] {
			t := (lambda - points[j-1][0]) / (points[j][0] - points[j-1][0])
			return points[j-1][1]*(1-t) + points[j][1]*t
		}
	}
	return points[len(points)-1][1]
}

// Gaussian basis reflectances and their precomputed XYZ under D65.
var (
	basisR     SPD
	basisG     SPD
	basisB     SPD
	basisXYZInv [3][3]float64
)

func init() {
	// Build Gaussian basis reflectances with IR/UV tails.
	// The visible-range Gaussians are extended with smooth tails that approximate
	// real material behavior: warm-colored materials have high near-IR reflectance,
	// while blue materials absorb IR. UV reflectance is generally low for all.
	for i := range basisR {
		lambda := Wavelength(i)

		// Visible-range Gaussians (same as before)
		rVis := math.Exp(-0.5 * math.Pow((lambda-600)/40, 2))
		gVis := math.Exp(-0.5 * math.Pow((lambda-540)/35, 2))
		bVis := math.Exp(-0.5 * math.Pow((lambda-450)/30, 2))

		// IR tails (>780nm): red materials stay reflective, green/blue decay
		var rIR, gIR, bIR float64
		if lambda > 780 {
			t := (lambda - 780) / 500 // normalized distance into IR
			rIR = 0.6 * math.Exp(-0.5*t*t)      // red: strong IR reflectance, slow decay
			gIR = 0.3 * math.Exp(-2.0*t*t)       // green: moderate IR, faster decay
			bIR = 0.05 * math.Exp(-3.0*t*t)      // blue: low IR reflectance
		}

		// UV tails (<380nm): most materials absorb UV
		var rUV, gUV, bUV float64
		if lambda < 380 {
			t := (380 - lambda) / 100
			rUV = 0.02 * math.Exp(-2.0*t*t)
			gUV = 0.02 * math.Exp(-2.0*t*t)
			bUV = 0.05 * math.Exp(-1.0*t*t) // blue pigments reflect slightly more UV
		}

		basisR[i] = rVis + rIR + rUV
		basisG[i] = gVis + gIR + gUV
		basisB[i] = bVis + bIR + bUV
	}

	// Compute XYZ of each basis under D65 illumination:
	//   XYZ_i = ∫ D65(λ) * basis_i(λ) * [x̄,ȳ,z̄](λ) dλ
	d65 := D65()
	var m [3][3]float64 // m[basis][xyz_component]
	bases := [3]*SPD{&basisR, &basisG, &basisB}
	for bi, basis := range bases {
		for k := 0; k < NumBands; k++ {
			lit := d65[k] * basis[k]
			m[bi][0] += lit * cieX[k] * LambdaStep
			m[bi][1] += lit * cieY[k] * LambdaStep
			m[bi][2] += lit * cieZ[k] * LambdaStep
		}
	}

	// We need the matrix that maps [cr, cg, cb] -> [X, Y, Z]:
	//   [X]   [m[0][0] m[1][0] m[2][0]] [cr]
	//   [Y] = [m[0][1] m[1][1] m[2][1]] [cg]
	//   [Z]   [m[0][2] m[1][2] m[2][2]] [cb]
	// So the forward matrix has basis XYZ as columns.
	var fwd [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fwd[i][j] = m[j][i] // transpose: column j = basis j's XYZ
		}
	}
	basisXYZInv = invert3x3(fwd)
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
