#include "scene.h"
#include <math.h>

Spd sky_eval(const SkyParams *sky, Vec3 dir) {
    switch (sky->type) {
    case SKY_NONE:
        return spd_zero();
    case SKY_UNIFORM:
        return sky->emission;
    case SKY_GRADIENT: {
        Vec3 unit = vec3_normalize(dir);
        /* Map y from [-1, 1] to [0, 1] */
        double t = 0.5 * (unit.y + 1.0);
        /* Lerp: bottom * (1-t) + top * t */
        Spd bottom_part = spd_scale(&sky->bottom, 1.0 - t);
        Spd top_part = spd_scale(&sky->top, t);
        return spd_add(&bottom_part, &top_part);
    }
    }
    return spd_zero();
}

int scene_intersect(const Scene *scene, Vec3 origin, Vec3 dir,
                    double t_min, double t_max,
                    Hit *hit, const Material **mat_out) {
    int found = 0;
    double closest = t_max;

    /* Static objects */
    for (int i = 0; i < scene->num_objects; i++) {
        Hit tmp;
        if (shape_intersect(&scene->objects[i].shape, origin, dir,
                            t_min, closest, &tmp)) {
            *hit = tmp;
            hit->source_velocity = VEC3_ZERO;
            *mat_out = &scene->objects[i].material;
            closest = tmp.t;
            found = 1;
        }
    }

    /* Moving objects — solve retarded time */
    for (int i = 0; i < scene->num_moving; i++) {
        const MovingObject *mo = &scene->moving_objects[i];

        /* Evaluate trajectory at the scene observation time to get
           the retarded position and velocity of the object. */
        Vec3 obj_pos = trajectory_eval(&mo->trajectory, scene->time);
        Vec3 obj_vel = trajectory_velocity(&mo->trajectory, scene->time);

        /* Attempt retarded time solve: find the emission time such that
           light from the object reaches the ray origin at scene->time. */
        double t_ret = 0;
        int solved = retarded_solve(&mo->trajectory, origin, scene->time, &t_ret);
        if (solved) {
            obj_pos = trajectory_eval(&mo->trajectory, t_ret);
            obj_vel = trajectory_velocity(&mo->trajectory, t_ret);
        }

        /* Translate ray into the object's local frame
           (object at obj_pos with same rotation as its shape transform) */
        Vec3 local_origin = vec3_sub(origin, obj_pos);

        /* Build a temporary shape without the position offset,
           since we've already accounted for it */
        Shape local_shape = mo->shape;
        if (local_shape.has_transform) {
            local_shape.position = VEC3_ZERO;
        }

        Hit tmp;
        if (shape_intersect(&local_shape, local_origin, dir,
                            t_min, closest, &tmp)) {
            /* Transform hit point back to world space */
            tmp.point = vec3_add(tmp.point, obj_pos);
            tmp.source_velocity = obj_vel;
            *hit = tmp;
            *mat_out = &mo->material;
            closest = tmp.t;
            found = 1;
        }
    }

    return found;
}
