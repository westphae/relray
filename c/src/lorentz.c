#include "lorentz.h"
#include <math.h>

double lorentz_gamma(Vec3 beta) {
    double b2 = vec3_length_sq(beta);
    if (b2 == 0.0) return 1.0;
    return 1.0 / sqrt(1.0 - b2);
}

AberrationResult lorentz_aberrate(Vec3 dir_obs, Vec3 beta) {
    double b2 = vec3_length_sq(beta);
    if (b2 == 0.0) {
        return (AberrationResult){ .dir = dir_obs, .doppler = 1.0 };
    }

    double gamma = 1.0 / sqrt(1.0 - b2);

    /* Photon propagation direction in observer frame */
    Vec3 p = vec3_neg(dir_obs);

    /* Null 4-wavevector in observer frame: k_obs = (1, p)
     * Lorentz boost to world frame (boost velocity = +beta):
     *   k_world^0 = gamma * (1 + beta . p)
     *   k_world_spatial = p + beta*gamma + (gamma-1)/b2 * (beta.p) * beta */
    double bdotp = vec3_dot(beta, p);

    double kw0 = gamma * (1.0 + bdotp);

    double factor = (gamma - 1.0) / b2 * bdotp;
    Vec3 kw_spatial = vec3_add(vec3_add(p, vec3_scale(beta, gamma)),
                               vec3_scale(beta, factor));

    /* Ray direction = negated photon propagation direction */
    Vec3 dir = vec3_normalize(vec3_neg(kw_spatial));
    double doppler = 1.0 / kw0;

    return (AberrationResult){ .dir = dir, .doppler = doppler };
}
