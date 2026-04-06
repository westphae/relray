use crate::cie_data::{CIE_X, CIE_Y, CIE_Z, D65_RAW};
use std::sync::LazyLock;

pub const NUM_BANDS: usize = 91;
pub const LAMBDA_MIN: f64 = 200.0;
#[allow(dead_code)]
pub const LAMBDA_MAX: f64 = 2000.0;
pub const LAMBDA_STEP: f64 = 20.0;
const VISIBLE_START: usize = 9;  // band index of 380nm: (380-200)/20
const VISIBLE_END: usize = 29;   // band index of 780nm: (780-200)/20

/// Spectral power distribution sampled at 20nm intervals from 200nm to 2000nm.
#[derive(Clone, Copy)]
pub struct Spd(pub [f64; NUM_BANDS]);

impl Default for Spd {
    fn default() -> Self {
        Spd([0.0; NUM_BANDS])
    }
}

impl Spd {
    /// Element-wise addition.
    pub fn add(self, other: Self) -> Self {
        let mut r = Spd::default();
        for i in 0..NUM_BANDS {
            r.0[i] = self.0[i] + other.0[i];
        }
        r
    }

    /// Element-wise multiplication.
    pub fn mul(self, other: Self) -> Self {
        let mut r = Spd::default();
        for i in 0..NUM_BANDS {
            r.0[i] = self.0[i] * other.0[i];
        }
        r
    }

    /// Scale all bands by a constant factor.
    pub fn scale(self, f: f64) -> Self {
        let mut r = Spd::default();
        for i in 0..NUM_BANDS {
            r.0[i] = self.0[i] * f;
        }
        r
    }

    /// Return a new SPD with wavelengths scaled by `factor`.
    /// factor > 1 = redshift (wavelengths increase), factor < 1 = blueshift.
    /// Uses linear interpolation for resampling.
    pub fn shift(self, factor: f64) -> Self {
        if factor == 1.0 {
            return self;
        }
        let mut r = Spd::default();
        for i in 0..NUM_BANDS {
            // What original wavelength maps to this band after shifting?
            // new_lambda = old_lambda * factor, so old_lambda = new_lambda / factor
            let orig_lambda = wavelength(i) / factor;
            let idx = band_index(orig_lambda);
            if idx < 0.0 || idx >= (NUM_BANDS - 1) as f64 {
                continue;
            }
            let lo = idx as usize;
            let frac = idx - lo as f64;
            r.0[i] = self.0[lo] * (1.0 - frac) + self.0[lo + 1] * frac;
        }
        r
    }

    /// Integrate the SPD against the CIE 1931 2-degree color matching functions.
    /// Only iterates over the visible range (380-780nm) since CIE values are zero outside.
    pub fn to_xyz(&self) -> (f64, f64, f64) {
        let mut x = 0.0;
        let mut y = 0.0;
        let mut z = 0.0;
        for i in VISIBLE_START..=VISIBLE_END {
            x += self.0[i] * CIE_X[i];
            y += self.0[i] * CIE_Y[i];
            z += self.0[i] * CIE_Z[i];
        }
        x *= LAMBDA_STEP;
        y *= LAMBDA_STEP;
        z *= LAMBDA_STEP;
        (x, y, z)
    }
}

/// Returns the wavelength in nm for band index `i`.
pub fn wavelength(i: usize) -> f64 {
    LAMBDA_MIN + i as f64 * LAMBDA_STEP
}

/// Returns the fractional band index for a wavelength.
pub fn band_index(lambda: f64) -> f64 {
    (lambda - LAMBDA_MIN) / LAMBDA_STEP
}

/// Convert CIE XYZ to linear sRGB (not gamma-corrected).
pub fn xyz_to_linear_rgb(x: f64, y: f64, z: f64) -> (f64, f64, f64) {
    // sRGB D65 matrix (IEC 61966-2-1)
    let r = 3.2406 * x - 1.5372 * y - 0.4986 * z;
    let g = -0.9689 * x + 1.8758 * y + 0.0415 * z;
    let b = 0.0557 * x - 0.2040 * y + 1.0570 * z;
    (r, g, b)
}

/// Convert CIE XYZ to 8-bit sRGB with gamma correction and clamping.
pub fn xyz_to_srgb(x: f64, y: f64, z: f64) -> (u8, u8, u8) {
    let (lr, lg, lb) = xyz_to_linear_rgb(x, y, z);
    (to_u8(srgb_gamma(lr)), to_u8(srgb_gamma(lg)), to_u8(srgb_gamma(lb)))
}

fn srgb_gamma(c: f64) -> f64 {
    if c <= 0.0031308 {
        12.92 * c
    } else {
        1.055 * c.powf(1.0 / 2.4) - 0.055
    }
}

fn to_u8(v: f64) -> u8 {
    (v * 255.0).clamp(0.0, 255.0).round() as u8
}

/// Returns the CIE standard illuminant D65, normalized so that Y=1.
pub fn d65() -> Spd {
    let mut sum = 0.0;
    for i in 0..NUM_BANDS {
        sum += D65_RAW[i] * CIE_Y[i];
    }
    sum *= LAMBDA_STEP;
    let mut s = Spd::default();
    for i in 0..NUM_BANDS {
        s.0[i] = D65_RAW[i] / sum;
    }
    s
}

/// Returns an SPD with all bands set to `v`.
pub fn constant(v: f64) -> Spd {
    Spd([v; NUM_BANDS])
}

/// Returns an SPD with energy concentrated at the given wavelength.
/// The energy is distributed across the two nearest bands via linear interpolation.
pub fn monochromatic(lambda: f64, power: f64) -> Spd {
    let mut s = Spd::default();
    let idx = band_index(lambda);
    if idx < 0.0 || idx >= (NUM_BANDS - 1) as f64 {
        return s;
    }
    let lo = idx as usize;
    let frac = idx - lo as f64;
    s.0[lo] = power * (1.0 - frac);
    if lo + 1 < NUM_BANDS {
        s.0[lo + 1] = power * frac;
    }
    s
}

/// Creates a reflectance SPD from linear sRGB values.
///
/// Uses three Gaussian basis reflectances calibrated so that under D65 illumination,
/// the round-trip through CIE XYZ -> sRGB reproduces the input colors correctly.
pub fn from_rgb(r: f64, g: f64, b: f64) -> Spd {
    let basis = &*RGB_BASIS;

    // Target XYZ under D65 illumination (sRGB forward matrix)
    let tx = 0.4124564 * r + 0.3575761 * g + 0.1804375 * b;
    let ty = 0.2126729 * r + 0.7151522 * g + 0.0721750 * b;
    let tz = 0.0193339 * r + 0.1191920 * g + 0.9503041 * b;

    // Solve for coefficients: c = basisXYZInv * [tx, ty, tz]
    let inv = &basis.xyz_inv;
    let cr = inv[0][0] * tx + inv[0][1] * ty + inv[0][2] * tz;
    let cg = inv[1][0] * tx + inv[1][1] * ty + inv[1][2] * tz;
    let cb = inv[2][0] * tx + inv[2][1] * ty + inv[2][2] * tz;

    let mut s = Spd::default();
    for i in 0..NUM_BANDS {
        let v = cr * basis.r.0[i] + cg * basis.g.0[i] + cb * basis.b.0[i];
        s.0[i] = if v < 0.0 { 0.0 } else { v };
    }
    s
}

/// Creates an SPD from a set of (wavelength_nm, value) pairs.
/// The pairs are linearly interpolated to fill all bands. Wavelengths outside the
/// given range clamp to the nearest endpoint value. Points must be sorted by wavelength.
pub fn from_reflectance_curve(points: &[[f64; 2]]) -> Spd {
    if points.is_empty() {
        return Spd::default();
    }
    let mut s = Spd::default();
    for i in 0..NUM_BANDS {
        let lambda = wavelength(i);
        s.0[i] = interpolate_points(points, lambda);
    }
    s
}

fn interpolate_points(points: &[[f64; 2]], lambda: f64) -> f64 {
    if lambda <= points[0][0] {
        return points[0][1];
    }
    if lambda >= points[points.len() - 1][0] {
        return points[points.len() - 1][1];
    }
    for j in 1..points.len() {
        if lambda <= points[j][0] {
            let t = (lambda - points[j - 1][0]) / (points[j][0] - points[j - 1][0]);
            return points[j - 1][1] * (1.0 - t) + points[j][1] * t;
        }
    }
    points[points.len() - 1][1]
}

// Planck constants
const PLANCK_H: f64 = 6.62607015e-34; // J*s
const PLANCK_C: f64 = 2.99792458e8; // m/s
const BOLTZ_K: f64 = 1.380649e-23; // J/K

/// Returns the spectral radiance of a blackbody at temperature `temp_k` (Kelvin),
/// normalized so that the Y component equals the given luminance.
pub fn blackbody(temp_k: f64, luminance: f64) -> Spd {
    let mut s = Spd::default();
    for i in 0..NUM_BANDS {
        let lambda_m = wavelength(i) * 1e-9; // nm to meters
        s.0[i] = planck_radiance(lambda_m, temp_k);
    }

    // Normalize to desired luminance (Y value)
    let (_, y, _) = s.to_xyz();
    if y > 0.0 {
        s = s.scale(luminance / y);
    }
    s
}

/// Planck radiance B(lambda, T) in W*sr^-1*m^-3.
fn planck_radiance(lambda_m: f64, temp_k: f64) -> f64 {
    let a = 2.0 * PLANCK_H * PLANCK_C * PLANCK_C / lambda_m.powi(5);
    let exponent = PLANCK_H * PLANCK_C / (lambda_m * BOLTZ_K * temp_k);
    if exponent > 500.0 {
        return 0.0;
    }
    a / (exponent.exp() - 1.0)
}

// --- Precomputed RGB basis SPDs and inverse matrix ---

struct RgbBasis {
    r: Spd,
    g: Spd,
    b: Spd,
    xyz_inv: [[f64; 3]; 3],
}

static RGB_BASIS: LazyLock<RgbBasis> = LazyLock::new(|| {
    let mut basis_r = Spd::default();
    let mut basis_g = Spd::default();
    let mut basis_b = Spd::default();

    for i in 0..NUM_BANDS {
        let lambda = wavelength(i);

        // Visible-range Gaussians
        let r_vis = (-0.5 * ((lambda - 600.0) / 40.0).powi(2)).exp();
        let g_vis = (-0.5 * ((lambda - 540.0) / 35.0).powi(2)).exp();
        let b_vis = (-0.5 * ((lambda - 450.0) / 30.0).powi(2)).exp();

        // IR tails (>780nm)
        let (r_ir, g_ir, b_ir) = if lambda > 780.0 {
            let t = (lambda - 780.0) / 500.0;
            (
                0.6 * (-0.5 * t * t).exp(),
                0.3 * (-2.0 * t * t).exp(),
                0.05 * (-3.0 * t * t).exp(),
            )
        } else {
            (0.0, 0.0, 0.0)
        };

        // UV tails (<380nm)
        let (r_uv, g_uv, b_uv) = if lambda < 380.0 {
            let t = (380.0 - lambda) / 100.0;
            (
                0.02 * (-2.0 * t * t).exp(),
                0.02 * (-2.0 * t * t).exp(),
                0.05 * (-1.0 * t * t).exp(),
            )
        } else {
            (0.0, 0.0, 0.0)
        };

        basis_r.0[i] = r_vis + r_ir + r_uv;
        basis_g.0[i] = g_vis + g_ir + g_uv;
        basis_b.0[i] = b_vis + b_ir + b_uv;
    }

    // Compute XYZ of each basis under D65 illumination
    let d65_spd = d65();
    let mut m = [[0.0_f64; 3]; 3]; // m[basis][xyz_component]
    let bases = [&basis_r, &basis_g, &basis_b];
    for (bi, basis) in bases.iter().enumerate() {
        for k in 0..NUM_BANDS {
            let lit = d65_spd.0[k] * basis.0[k];
            m[bi][0] += lit * CIE_X[k] * LAMBDA_STEP;
            m[bi][1] += lit * CIE_Y[k] * LAMBDA_STEP;
            m[bi][2] += lit * CIE_Z[k] * LAMBDA_STEP;
        }
    }

    // Forward matrix: columns are basis XYZ
    let mut fwd = [[0.0_f64; 3]; 3];
    for i in 0..3 {
        for j in 0..3 {
            fwd[i][j] = m[j][i]; // transpose: column j = basis j's XYZ
        }
    }

    let xyz_inv = invert3x3(fwd);

    RgbBasis {
        r: basis_r,
        g: basis_g,
        b: basis_b,
        xyz_inv,
    }
});

fn invert3x3(m: [[f64; 3]; 3]) -> [[f64; 3]; 3] {
    let (a, b, c) = (m[0][0], m[0][1], m[0][2]);
    let (d, e, f) = (m[1][0], m[1][1], m[1][2]);
    let (g, h, k) = (m[2][0], m[2][1], m[2][2]);

    let det = a * (e * k - f * h) - b * (d * k - f * g) + c * (d * h - e * g);

    let mut inv = [[0.0_f64; 3]; 3];
    inv[0][0] = (e * k - f * h) / det;
    inv[0][1] = (c * h - b * k) / det;
    inv[0][2] = (b * f - c * e) / det;
    inv[1][0] = (f * g - d * k) / det;
    inv[1][1] = (a * k - c * g) / det;
    inv[1][2] = (c * d - a * f) / det;
    inv[2][0] = (d * h - e * g) / det;
    inv[2][1] = (b * g - a * h) / det;
    inv[2][2] = (a * e - b * d) / det;
    inv
}

#[cfg(test)]
mod tests {
    use super::*;

    const EPS: f64 = 1e-6;

    #[test]
    fn test_d65_normalized_y() {
        let d = d65();
        let (_, y, _) = d.to_xyz();
        assert!(
            (y - 1.0).abs() < 0.01,
            "D65 Y = {y}, want ~1.0"
        );
    }

    #[test]
    fn test_d65_to_srgb() {
        let d = d65();
        let (x, y, z) = d.to_xyz();
        let (r, g, b) = xyz_to_srgb(x, y, z);
        assert!(
            r >= 200 && g >= 200 && b >= 200,
            "D65 sRGB = ({r}, {g}, {b}), expected near-white"
        );
    }

    #[test]
    fn test_shift_identity() {
        let d = d65();
        let shifted = d.shift(1.0);
        for i in 0..NUM_BANDS {
            assert!(
                (shifted.0[i] - d.0[i]).abs() < EPS,
                "Shift(1.0) changed band {i}: {} -> {}",
                d.0[i],
                shifted.0[i]
            );
        }
    }

    #[test]
    fn test_shift_redshift() {
        // Monochromatic 550nm shifted by factor 2 -> 1100nm (in extended range)
        let m = monochromatic(550.0, 1.0);
        let shifted = m.shift(2.0);
        let (_, y, _) = shifted.to_xyz();
        assert!(
            y < EPS,
            "550nm redshifted to 1100nm should have zero visible luminance, got Y={y}"
        );
        let total: f64 = shifted.0.iter().sum();
        assert!(
            total > EPS,
            "550nm redshifted to 1100nm should have IR energy in extended range"
        );
    }

    #[test]
    fn test_shift_blueshift() {
        // Monochromatic 600nm shifted by factor 0.5 -> 300nm (in extended range)
        let m = monochromatic(600.0, 1.0);
        let shifted = m.shift(0.5);
        let (_, y, _) = shifted.to_xyz();
        assert!(
            y < EPS,
            "600nm blueshifted to 300nm should have zero visible luminance, got Y={y}"
        );
        let total: f64 = shifted.0.iter().sum();
        assert!(
            total > EPS,
            "600nm blueshifted to 300nm should have UV energy in extended range"
        );
    }

    #[test]
    fn test_shift_ir_into_visible() {
        // Monochromatic 1000nm (invisible IR) blueshifted by factor 0.5 -> 500nm (visible green)
        let m = monochromatic(1000.0, 1.0);
        let (_, y, _) = m.to_xyz();
        assert!(y < EPS, "1000nm should be invisible, got Y={y}");

        let shifted = m.shift(0.5);
        let (_, y2, _) = shifted.to_xyz();
        assert!(
            y2 > 0.01,
            "1000nm blueshifted to 500nm should be visible, got Y={y2}"
        );
    }

    #[test]
    fn test_monochromatic() {
        let m = monochromatic(550.0, 1.0);
        let idx = band_index(550.0) as usize;
        assert!(
            m.0[idx] > 0.0,
            "monochromatic 550nm should have power at band {idx}"
        );
        assert!(
            m.0[0] == 0.0,
            "monochromatic 550nm should be zero at 200nm, got {}",
            m.0[0]
        );
    }

    #[test]
    fn test_blackbody_sunlike() {
        let bb = blackbody(5778.0, 1.0);
        let (_, y, _) = bb.to_xyz();
        assert!(
            (y - 1.0).abs() < 0.01,
            "Blackbody(5778, 1.0) Y = {y}, want ~1.0"
        );
        assert!(
            bb.0[0] > 0.0 && bb.0[40] > 0.0 && bb.0[80] > 0.0,
            "Blackbody should have power across entire visible range"
        );
    }

    #[test]
    fn test_blackbody_hot() {
        let bb = blackbody(10000.0, 1.0);
        let (x, _, z) = bb.to_xyz();
        assert!(
            z > x,
            "10000K blackbody should be blue-biased: X={x} Z={z}"
        );
    }

    #[test]
    fn test_blackbody_cool() {
        let bb = blackbody(3000.0, 1.0);
        let (x, _, z) = bb.to_xyz();
        assert!(
            x > z,
            "3000K blackbody should be red-biased: X={x} Z={z}"
        );
    }

    #[test]
    fn test_from_rgb_red() {
        let r = from_rgb(1.0, 0.0, 0.0);
        let (x, y, z) = r.to_xyz();
        let (rr, gg, bb) = xyz_to_linear_rgb(x, y, z);
        assert!(
            rr > gg && rr > bb,
            "FromRGB(1,0,0) should have red dominant, got linear RGB ({rr}, {gg}, {bb})"
        );
    }
}
