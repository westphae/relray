#include "material.h"
#include <math.h>

Vec3 random_unit_vec(Rng *rng) {
    for (;;) {
        Vec3 v = vec3(2.0 * rng_float64(rng) - 1.0,
                      2.0 * rng_float64(rng) - 1.0,
                      2.0 * rng_float64(rng) - 1.0);
        double l2 = vec3_length_sq(v);
        if (l2 > 1e-6 && l2 <= 1.0)
            return vec3_scale(v, 1.0 / sqrt(l2));
    }
}

static double clamp_d(double v, double lo, double hi) {
    if (v < lo) return lo;
    if (v > hi) return hi;
    return v;
}

static double schlick(double cosine, double eta_ratio) {
    double r0 = (1.0 - eta_ratio) / (1.0 + eta_ratio);
    r0 = r0 * r0;
    return r0 + (1.0 - r0) * pow(1.0 - cosine, 5.0);
}

static Vec3 refract_vec(Vec3 uv, Vec3 n, double eta_ratio) {
    double cos_theta = fmin(-vec3_dot(uv, n), 1.0);
    Vec3 r_perp = vec3_scale(vec3_add(uv, vec3_scale(n, cos_theta)), eta_ratio);
    Vec3 r_parallel = vec3_scale(n, -sqrt(fabs(1.0 - vec3_length_sq(r_perp))));
    return vec3_add(r_perp, r_parallel);
}

static ScatterResult scatter_diffuse(const Spd *reflectance,
                                     const Hit *hit, Rng *rng) {
    Vec3 scattered = vec3_add(hit->normal, random_unit_vec(rng));
    if (vec3_length_sq(scattered) < 1e-12)
        scattered = hit->normal;
    return (ScatterResult){
        .scattered = 1,
        .out_dir = vec3_normalize(scattered),
        .reflectance = *reflectance
    };
}

static ScatterResult scatter_mirror(const Spd *reflectance,
                                    Vec3 in_dir, const Hit *hit) {
    Vec3 reflected = vec3_reflect(in_dir, hit->normal);
    return (ScatterResult){
        .scattered = 1,
        .out_dir = vec3_normalize(reflected),
        .reflectance = *reflectance
    };
}

static ScatterResult scatter_glass(double ior, const Spd *tint,
                                   Vec3 in_dir, const Hit *hit, Rng *rng) {
    double eta_ratio = hit->front_face ? (1.0 / ior) : ior;
    Vec3 unit_dir = vec3_normalize(in_dir);
    double cos_i = fmin(-vec3_dot(unit_dir, hit->normal), 1.0);
    double sin2_t = eta_ratio * eta_ratio * (1.0 - cos_i * cos_i);

    /* Total internal reflection */
    if (sin2_t > 1.0) {
        Vec3 reflected = vec3_reflect(unit_dir, hit->normal);
        return (ScatterResult){
            .scattered = 1,
            .out_dir = vec3_normalize(reflected),
            .reflectance = *tint
        };
    }

    /* Schlick approximation for Fresnel */
    double refl = schlick(cos_i, eta_ratio);
    if (rng_float64(rng) < refl) {
        Vec3 reflected = vec3_reflect(unit_dir, hit->normal);
        return (ScatterResult){
            .scattered = 1,
            .out_dir = vec3_normalize(reflected),
            .reflectance = *tint
        };
    }

    Vec3 refracted = refract_vec(unit_dir, hit->normal, eta_ratio);
    return (ScatterResult){
        .scattered = 1,
        .out_dir = vec3_normalize(refracted),
        .reflectance = *tint
    };
}

static Spd checker_reflectance_at(const Spd *even, const Spd *odd,
                                  double scale, Vec3 point) {
    double inv = 1.0 / scale;
    int ix = (int)floor(point.x * inv);
    int iz = (int)floor(point.z * inv);
    return ((ix + iz) % 2 == 0) ? *even : *odd;
}

static Spd checker_sphere_reflectance_at(const Spd *even, const Spd *odd,
                                         int num_squares, Vec3 normal) {
    int n = num_squares;
    if (n <= 0) n = 8;

    double lat = asin(clamp_d(normal.y, -1.0, 1.0));
    double lon = atan2(normal.z, normal.x);

    int lat_div = (int)floor(lat * (double)n / M_PI);
    int lon_div = (int)floor(lon * (double)n / M_PI);

    return ((lat_div + lon_div) % 2 == 0) ? *even : *odd;
}

ScatterResult material_scatter(const Material *mat, Vec3 in_dir,
                               const Hit *hit, Rng *rng) {
    switch (mat->type) {
    case MAT_DIFFUSE:
        return scatter_diffuse(&mat->diffuse.reflectance, hit, rng);

    case MAT_MIRROR:
        return scatter_mirror(&mat->mirror.reflectance, in_dir, hit);

    case MAT_GLASS:
        return scatter_glass(mat->glass.ior, &mat->glass.tint,
                             in_dir, hit, rng);

    case MAT_CHECKER: {
        Spd refl = checker_reflectance_at(&mat->checker.even,
                                          &mat->checker.odd,
                                          mat->checker.scale,
                                          hit->point);
        return scatter_diffuse(&refl, hit, rng);
    }

    case MAT_CHECKER_SPHERE: {
        Spd refl = checker_sphere_reflectance_at(&mat->checker_sphere.even,
                                                 &mat->checker_sphere.odd,
                                                 mat->checker_sphere.num_squares,
                                                 hit->normal);
        return scatter_diffuse(&refl, hit, rng);
    }
    }

    /* Unreachable, but silence compiler */
    return (ScatterResult){.scattered = 0};
}

Spd material_emitted(const Material *mat, const Hit *hit) {
    (void)mat;
    (void)hit;
    return spd_zero();
}
