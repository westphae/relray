#ifndef SCENE_H
#define SCENE_H

#include "shape.h"
#include "material.h"
#include "spd.h"
#include "retarded.h"

typedef struct { Vec3 position; Spd emission; } Light;

typedef struct {
    Shape shape;
    Material material;
} Object;

typedef struct {
    Shape shape;
    Material material;
    TrajectoryParams trajectory;
} MovingObject;

typedef enum { SKY_NONE, SKY_UNIFORM, SKY_GRADIENT } SkyType;

typedef struct {
    SkyType type;
    Spd top, bottom;   /* for gradient */
    Spd emission;      /* for uniform */
} SkyParams;

Spd sky_eval(const SkyParams *sky, Vec3 dir);

typedef struct {
    char name[64];
    Object *objects;
    int num_objects;
    MovingObject *moving_objects;
    int num_moving;
    Light *lights;
    int num_lights;
    SkyParams sky;
    double time;
} Scene;

/* Returns 1 if hit, fills hit and mat_out */
int scene_intersect(const Scene *scene, Vec3 origin, Vec3 dir,
                    double t_min, double t_max,
                    Hit *hit, const Material **mat_out);

#endif
