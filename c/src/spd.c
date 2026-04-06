#include "spd.h"
#include "cie_data.h"
#include <math.h>
#include <string.h>

/* --- Arithmetic (return new SPD) --- */

Spd spd_add(const Spd *a, const Spd *b) {
    Spd r;
    for (int i = 0; i < NUM_BANDS; i++)
        r.data[i] = a->data[i] + b->data[i];
    return r;
}

Spd spd_mul(const Spd *a, const Spd *b) {
    Spd r;
    for (int i = 0; i < NUM_BANDS; i++)
        r.data[i] = a->data[i] * b->data[i];
    return r;
}

Spd spd_scale(const Spd *a, double f) {
    Spd r;
    for (int i = 0; i < NUM_BANDS; i++)
        r.data[i] = a->data[i] * f;
    return r;
}

/* --- In-place (modify first arg) --- */

void spd_add_inplace(Spd *__restrict__ dst, const Spd *__restrict__ src) {
    for (int i = 0; i < NUM_BANDS; i++)
        dst->data[i] += src->data[i];
}

void spd_mul_inplace(Spd *__restrict__ dst, const Spd *__restrict__ src) {
    for (int i = 0; i < NUM_BANDS; i++)
        dst->data[i] *= src->data[i];
}

void spd_scale_inplace(Spd *dst, double f) {
    for (int i = 0; i < NUM_BANDS; i++)
        dst->data[i] *= f;
}

/* --- Shift (wavelength rescaling with linear interpolation) --- */

Spd spd_shift(const Spd *a, double factor) {
    if (factor == 1.0) return *a;

    Spd r;
    memset(&r, 0, sizeof(r));

    double inv_factor = 1.0 / factor;
    double start_idx = (LAMBDA_MIN * inv_factor - LAMBDA_MIN) / LAMBDA_STEP;
    double step = inv_factor;
    double max_idx = (double)(NUM_BANDS - 1);

    double idx = start_idx;
    for (int i = 0; i < NUM_BANDS; i++) {
        if (idx >= 0.0 && idx < max_idx) {
            int lo = (int)idx;
            double frac = idx - (double)lo;
            r.data[i] = fma(a->data[lo + 1] - a->data[lo], frac, a->data[lo]);
        }
        idx += step;
    }
    return r;
}

/* --- Color conversion --- */

void spd_to_xyz(const Spd *s, double *x, double *y, double *z) {
    double sx = 0.0, sy = 0.0, sz = 0.0;
    for (int i = VISIBLE_START; i <= VISIBLE_END; i++) {
        sx += s->data[i] * cie_x[i];
        sy += s->data[i] * cie_y[i];
        sz += s->data[i] * cie_z[i];
    }
    *x = sx * LAMBDA_STEP;
    *y = sy * LAMBDA_STEP;
    *z = sz * LAMBDA_STEP;
}

static double srgb_gamma(double c) {
    if (c <= 0.0031308)
        return 12.92 * c;
    return 1.055 * pow(c, 1.0 / 2.4) - 0.055;
}

static uint8_t to_uint8(double v) {
    double c = v * 255.0;
    if (c < 0.0) return 0;
    if (c > 255.0) return 255;
    return (uint8_t)(c + 0.5);
}

void xyz_to_srgb(double x, double y, double z, uint8_t *r, uint8_t *g, uint8_t *b) {
    /* sRGB D65 matrix (IEC 61966-2-1) */
    double lr =  3.2406 * x - 1.5372 * y - 0.4986 * z;
    double lg = -0.9689 * x + 1.8758 * y + 0.0415 * z;
    double lb =  0.0557 * x - 0.2040 * y + 1.0570 * z;

    *r = to_uint8(srgb_gamma(lr));
    *g = to_uint8(srgb_gamma(lg));
    *b = to_uint8(srgb_gamma(lb));
}

/* --- Constructors --- */

Spd spd_zero(void) {
    Spd s;
    memset(&s, 0, sizeof(s));
    return s;
}

Spd spd_constant(double v) {
    Spd s;
    for (int i = 0; i < NUM_BANDS; i++)
        s.data[i] = v;
    return s;
}

/* D65: normalized so Y=1. Cached after first call. */
Spd spd_d65(void) {
    static Spd cached;
    static int ready = 0;
    if (ready) return cached;

    double sum = 0.0;
    for (int i = 0; i < NUM_BANDS; i++)
        sum += d65_raw[i] * cie_y[i];
    sum *= LAMBDA_STEP;

    for (int i = 0; i < NUM_BANDS; i++)
        cached.data[i] = d65_raw[i] / sum;

    ready = 1;
    return cached;
}

Spd spd_blackbody(double temp_k, double luminance) {
    /* Planck's law: B(lambda, T) = (2hc^2 / lambda^5) / (exp(hc / (lambda*k*T)) - 1)
     * We compute relative spectral shape, then scale so Y = luminance. */
    static const double h = 6.62607015e-34;
    static const double c = 2.99792458e8;
    static const double k = 1.380649e-23;

    Spd s;
    for (int i = 0; i < NUM_BANDS; i++) {
        double lambda_m = spd_wavelength(i) * 1e-9; /* nm -> m */
        double l5 = lambda_m * lambda_m * lambda_m * lambda_m * lambda_m;
        double exponent = (h * c) / (lambda_m * k * temp_k);
        s.data[i] = (2.0 * h * c * c) / (l5 * (exp(exponent) - 1.0));
    }

    /* Normalize to desired luminance via Y integral */
    double y_sum = 0.0;
    for (int i = VISIBLE_START; i <= VISIBLE_END; i++)
        y_sum += s.data[i] * cie_y[i];
    y_sum *= LAMBDA_STEP;

    if (y_sum > 0.0) {
        double scale = luminance / y_sum;
        for (int i = 0; i < NUM_BANDS; i++)
            s.data[i] *= scale;
    }

    return s;
}

Spd spd_monochromatic(double lambda, double power) {
    Spd s;
    memset(&s, 0, sizeof(s));
    double idx = spd_band_index(lambda);
    if (idx < 0.0 || idx >= (double)(NUM_BANDS - 1))
        return s;
    int lo = (int)idx;
    double frac = idx - (double)lo;
    s.data[lo] = power * (1.0 - frac);
    if (lo + 1 < NUM_BANDS)
        s.data[lo + 1] = power * frac;
    return s;
}

/* --- FromRGB: Gaussian basis reflectances calibrated under D65 --- */

static Spd basis_r, basis_g, basis_b;
static double basis_xyz_inv[3][3];
static int from_rgb_ready = 0;

static void invert3x3(const double m[3][3], double inv[3][3]) {
    double a = m[0][0], b = m[0][1], c = m[0][2];
    double d = m[1][0], e = m[1][1], f = m[1][2];
    double g = m[2][0], h = m[2][1], k = m[2][2];

    double det = a*(e*k - f*h) - b*(d*k - f*g) + c*(d*h - e*g);

    inv[0][0] = (e*k - f*h) / det;
    inv[0][1] = (c*h - b*k) / det;
    inv[0][2] = (b*f - c*e) / det;
    inv[1][0] = (f*g - d*k) / det;
    inv[1][1] = (a*k - c*g) / det;
    inv[1][2] = (c*d - a*f) / det;
    inv[2][0] = (d*h - e*g) / det;
    inv[2][1] = (b*g - a*h) / det;
    inv[2][2] = (a*e - b*d) / det;
}

static void init_from_rgb(void) {
    if (from_rgb_ready) return;

    /* Build Gaussian basis reflectances with IR/UV tails */
    for (int i = 0; i < NUM_BANDS; i++) {
        double lambda = spd_wavelength(i);

        /* Visible-range Gaussians */
        double r_vis = exp(-0.5 * pow((lambda - 600.0) / 40.0, 2));
        double g_vis = exp(-0.5 * pow((lambda - 540.0) / 35.0, 2));
        double b_vis = exp(-0.5 * pow((lambda - 450.0) / 30.0, 2));

        /* IR tails (>780nm) */
        double r_ir = 0.0, g_ir = 0.0, b_ir = 0.0;
        if (lambda > 780.0) {
            double t = (lambda - 780.0) / 500.0;
            r_ir = 0.6  * exp(-0.5 * t * t);
            g_ir = 0.3  * exp(-2.0 * t * t);
            b_ir = 0.05 * exp(-3.0 * t * t);
        }

        /* UV tails (<380nm) */
        double r_uv = 0.0, g_uv = 0.0, b_uv = 0.0;
        if (lambda < 380.0) {
            double t = (380.0 - lambda) / 100.0;
            r_uv = 0.02 * exp(-2.0 * t * t);
            g_uv = 0.02 * exp(-2.0 * t * t);
            b_uv = 0.05 * exp(-1.0 * t * t);
        }

        basis_r.data[i] = r_vis + r_ir + r_uv;
        basis_g.data[i] = g_vis + g_ir + g_uv;
        basis_b.data[i] = b_vis + b_ir + b_uv;
    }

    /* Compute XYZ of each basis under D65 illumination */
    Spd d65 = spd_d65();
    double m[3][3] = {{0}};  /* m[basis][xyz_component] */
    const Spd *bases[3] = { &basis_r, &basis_g, &basis_b };

    for (int bi = 0; bi < 3; bi++) {
        for (int k = 0; k < NUM_BANDS; k++) {
            double lit = d65.data[k] * bases[bi]->data[k];
            m[bi][0] += lit * cie_x[k] * LAMBDA_STEP;
            m[bi][1] += lit * cie_y[k] * LAMBDA_STEP;
            m[bi][2] += lit * cie_z[k] * LAMBDA_STEP;
        }
    }

    /* Forward matrix: columns are basis XYZ vectors */
    double fwd[3][3];
    for (int i = 0; i < 3; i++)
        for (int j = 0; j < 3; j++)
            fwd[i][j] = m[j][i];

    invert3x3(fwd, basis_xyz_inv);
    from_rgb_ready = 1;
}

Spd spd_from_rgb(double r, double g, double b) {
    init_from_rgb();

    /* Target XYZ under D65 illumination (sRGB to XYZ matrix) */
    double tx = 0.4124564 * r + 0.3575761 * g + 0.1804375 * b;
    double ty = 0.2126729 * r + 0.7151522 * g + 0.0721750 * b;
    double tz = 0.0193339 * r + 0.1191920 * g + 0.9503041 * b;

    /* Solve for coefficients: c = basisXYZInv * [tx, ty, tz] */
    double cr = basis_xyz_inv[0][0]*tx + basis_xyz_inv[0][1]*ty + basis_xyz_inv[0][2]*tz;
    double cg = basis_xyz_inv[1][0]*tx + basis_xyz_inv[1][1]*ty + basis_xyz_inv[1][2]*tz;
    double cb = basis_xyz_inv[2][0]*tx + basis_xyz_inv[2][1]*ty + basis_xyz_inv[2][2]*tz;

    Spd s;
    for (int i = 0; i < NUM_BANDS; i++) {
        double v = cr * basis_r.data[i] + cg * basis_g.data[i] + cb * basis_b.data[i];
        s.data[i] = (v < 0.0) ? 0.0 : v;
    }
    return s;
}

/* --- FromReflectanceCurve --- */

static double interpolate_points(const double points[][2], int n, double lambda) {
    if (lambda <= points[0][0])
        return points[0][1];
    if (lambda >= points[n - 1][0])
        return points[n - 1][1];
    for (int j = 1; j < n; j++) {
        if (lambda <= points[j][0]) {
            double t = (lambda - points[j - 1][0]) / (points[j][0] - points[j - 1][0]);
            return points[j - 1][1] * (1.0 - t) + points[j][1] * t;
        }
    }
    return points[n - 1][1];
}

Spd spd_from_reflectance_curve(const double points[][2], int num_points) {
    Spd s;
    memset(&s, 0, sizeof(s));
    if (num_points == 0)
        return s;
    for (int i = 0; i < NUM_BANDS; i++)
        s.data[i] = interpolate_points(points, num_points, spd_wavelength(i));
    return s;
}
