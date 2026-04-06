#include "retarded.h"
#include <math.h>

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

Vec3 trajectory_eval(const TrajectoryParams *t, double time) {
    switch (t->type) {
    case TRAJ_STATIC:
        return t->static_traj.position;
    case TRAJ_LINEAR:
        return vec3_add(t->linear.start, vec3_scale(t->linear.velocity, time));
    case TRAJ_ORBIT: {
        double angle = 2.0 * M_PI * time / t->orbit.period;
        double c = cos(angle), s = sin(angle);
        Vec3 center = t->orbit.center;
        double r = t->orbit.radius;
        switch (t->orbit.axis) {
        case 0: /* x-axis: orbit in YZ plane */
            return vec3(center.x, center.y + r * c, center.z + r * s);
        case 2: /* z-axis: orbit in XY plane */
            return vec3(center.x + r * c, center.y + r * s, center.z);
        default: /* y-axis (1): orbit in XZ plane */
            return vec3(center.x + r * c, center.y, center.z + r * s);
        }
    }
    }
    return VEC3_ZERO;
}

Vec3 trajectory_velocity(const TrajectoryParams *t, double time) {
    switch (t->type) {
    case TRAJ_STATIC:
        return VEC3_ZERO;
    case TRAJ_LINEAR:
        return t->linear.velocity;
    case TRAJ_ORBIT: {
        double omega = 2.0 * M_PI / t->orbit.period;
        double angle = omega * time;
        double c = cos(angle), s = sin(angle);
        double rw = t->orbit.radius * omega;
        switch (t->orbit.axis) {
        case 0: /* x-axis: orbit in YZ plane */
            return vec3(0, -rw * s, rw * c);
        case 2: /* z-axis: orbit in XY plane */
            return vec3(-rw * s, rw * c, 0);
        default: /* y-axis (1): orbit in XZ plane */
            return vec3(-rw * s, 0, rw * c);
        }
    }
    }
    return VEC3_ZERO;
}

int retarded_solve(const TrajectoryParams *traj, Vec3 obs_pos,
                   double t_obs, double *t_emit_out) {
    /* Initial guess: assume object is at its current position */
    Vec3 pos0 = trajectory_eval(traj, t_obs);
    double dist0 = vec3_length(vec3_sub(obs_pos, pos0));
    double t_emit = t_obs - dist0 / SPEED_OF_LIGHT;

    const int max_iter = 50;
    const double tol = 1e-10;

    for (int i = 0; i < max_iter; i++) {
        Vec3 obj_pos = trajectory_eval(traj, t_emit);
        Vec3 delta = vec3_sub(obs_pos, obj_pos);
        double dist = vec3_length(delta);
        double time_diff = t_obs - t_emit;

        double f = dist * dist - SPEED_OF_LIGHT * SPEED_OF_LIGHT * time_diff * time_diff;

        if (fabs(f) < tol) {
            *t_emit_out = t_emit;
            return 1;
        }

        /* Analytical velocity */
        Vec3 vel = trajectory_velocity(traj, t_emit);
        double fp = -2.0 * vec3_dot(delta, vel) +
                    2.0 * SPEED_OF_LIGHT * SPEED_OF_LIGHT * time_diff;

        if (fabs(fp) < 1e-20)
            break;

        t_emit -= f / fp;

        if (t_emit > t_obs)
            t_emit = t_obs - 1e-6;
    }

    /* Check convergence */
    Vec3 obj_pos = trajectory_eval(traj, t_emit);
    double dist = vec3_length(vec3_sub(obs_pos, obj_pos));
    double time_diff = t_obs - t_emit;
    double residual = fabs(dist - SPEED_OF_LIGHT * time_diff);
    if (residual < 1e-6) {
        *t_emit_out = t_emit;
        return 1;
    }
    return 0;
}
