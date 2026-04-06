#ifndef MATERIAL_H
#define MATERIAL_H

#include "spd.h"
#include "shape.h"
#include "rng.h"

typedef struct {
    int scattered;
    Vec3 out_dir;
    Spd reflectance;
} ScatterResult;

typedef enum {
    MAT_DIFFUSE, MAT_MIRROR, MAT_GLASS, MAT_CHECKER, MAT_CHECKER_SPHERE
} MaterialType;

typedef struct {
    MaterialType type;
    union {
        struct { Spd reflectance; } diffuse;
        struct { Spd reflectance; } mirror;
        struct { double ior; Spd tint; } glass;
        struct { Spd even, odd; double scale; } checker;
        struct { Spd even, odd; int num_squares; } checker_sphere;
    };
} Material;

ScatterResult material_scatter(const Material *mat, Vec3 in_dir,
                               const Hit *hit, Rng *rng);
Spd material_emitted(const Material *mat, const Hit *hit);
Vec3 random_unit_vec(Rng *rng);

#endif
