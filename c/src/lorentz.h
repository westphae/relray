#ifndef LORENTZ_H
#define LORENTZ_H

#include "vec.h"

typedef struct {
    Vec3 dir;       /* world-frame ray direction */
    double doppler; /* f_obs / f_emit */
} AberrationResult;

double lorentz_gamma(Vec3 beta);
AberrationResult lorentz_aberrate(Vec3 dir_obs, Vec3 beta);

#endif
