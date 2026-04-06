#ifndef SHAPE_H
#define SHAPE_H

#include "vec.h"

typedef struct {
    double t;
    Vec3 point;
    Vec3 normal;
    int front_face;
    Vec3 source_velocity;
} Hit;

void hit_set_face_normal(Hit *h, Vec3 ray_dir, Vec3 outward_normal);

typedef enum {
    SHAPE_SPHERE, SHAPE_PLANE, SHAPE_BOX, SHAPE_CYLINDER,
    SHAPE_CONE, SHAPE_DISK, SHAPE_TRIANGLE, SHAPE_PYRAMID
} ShapeType;

#define MAX_PYRAMID_FACES 32

typedef struct {
    ShapeType type;
    union {
        struct { double radius; } sphere;
        /* plane has no params (XZ at Y=0, normal +Y) */
        struct { Vec3 size; } box_shape;
        struct { double radius, height; } cylinder;
        struct { double radius, height; } cone;
        struct { double radius; } disk;
        struct { Vec3 v0, v1, v2; } triangle;
        struct { double base_radius, height; int sides; } pyramid;
    };
    /* Transform (applied to all shapes) */
    Vec3 position;
    Mat3 rotation;
    Mat3 inv_rot;
    int has_transform;
} Shape;

void shape_set_transform(Shape *s, Vec3 position, Mat3 rotation);

/* Returns 1 if hit, fills h */
int shape_intersect(const Shape *s, Vec3 origin, Vec3 dir,
                    double t_min, double t_max, Hit *h);

#endif
