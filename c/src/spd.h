#ifndef SPD_H
#define SPD_H

#include "vec.h"
#include <stdint.h>

#define NUM_BANDS 91
#define LAMBDA_MIN 200.0
#define LAMBDA_MAX 2000.0
#define LAMBDA_STEP 20.0
#define VISIBLE_START 9
#define VISIBLE_END 29

typedef struct { double data[NUM_BANDS]; } Spd;

/* Arithmetic (return new SPD) */
Spd spd_add(const Spd *a, const Spd *b);
Spd spd_mul(const Spd *a, const Spd *b);
Spd spd_scale(const Spd *a, double f);
Spd spd_shift(const Spd *a, double factor);

/* In-place (modify first arg) */
void spd_add_inplace(Spd *__restrict__ dst, const Spd *__restrict__ src);
void spd_mul_inplace(Spd *__restrict__ dst, const Spd *__restrict__ src);
void spd_scale_inplace(Spd *dst, double f);

/* Color conversion */
void spd_to_xyz(const Spd *s, double *x, double *y, double *z);
void xyz_to_srgb(double x, double y, double z, uint8_t *r, uint8_t *g, uint8_t *b);

/* Constructors */
Spd spd_zero(void);
Spd spd_constant(double v);
Spd spd_d65(void);
Spd spd_blackbody(double temp_k, double luminance);
Spd spd_from_rgb(double r, double g, double b);
Spd spd_monochromatic(double lambda, double power);
Spd spd_from_reflectance_curve(const double points[][2], int num_points);

static inline double spd_wavelength(int i) { return LAMBDA_MIN + i * LAMBDA_STEP; }
static inline double spd_band_index(double lambda) { return (lambda - LAMBDA_MIN) / LAMBDA_STEP; }

#endif
