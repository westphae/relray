#include "camera.h"
#include <math.h>

void camera_init(Camera *cam) {
    double theta = cam->vfov * M_PI / 180.0;
    cam->half_h = tan(theta / 2.0);
    cam->half_w = cam->half_h * cam->aspect;

    cam->w = vec3_normalize(vec3_sub(cam->position, cam->look_at));
    cam->u = vec3_normalize(vec3_cross(cam->up, cam->w));
    cam->v = vec3_cross(cam->w, cam->u);
}

Vec3 camera_ray_dir(const Camera *cam, double s, double t) {
    double x = (2.0 * s - 1.0) * cam->half_w;
    double y = (2.0 * t - 1.0) * cam->half_h;
    Vec3 dir = vec3_sub(vec3_add(vec3_scale(cam->u, x), vec3_scale(cam->v, y)), cam->w);
    return vec3_normalize(dir);
}
