#ifndef CAMERA_H
#define CAMERA_H

#include "vec.h"

typedef struct {
    Vec3 position;
    Vec3 look_at;
    Vec3 up;
    double vfov;    // vertical FOV in degrees
    double aspect;  // width / height
    Vec3 beta;      // velocity as fraction of c

    // Computed by camera_init()
    Vec3 u, v, w;
    double half_w, half_h;
} Camera;

void camera_init(Camera *cam);
Vec3 camera_ray_dir(const Camera *cam, double s, double t);

#endif
