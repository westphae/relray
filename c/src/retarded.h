#ifndef RETARDED_H
#define RETARDED_H

#include "vec.h"

#define SPEED_OF_LIGHT 1.0

/* Trajectory types (parameterized, no closures) */
typedef enum { TRAJ_STATIC, TRAJ_LINEAR, TRAJ_ORBIT } TrajectoryType;

typedef struct {
    TrajectoryType type;
    union {
        struct { Vec3 position; } static_traj;
        struct { Vec3 start, velocity; } linear;
        struct { Vec3 center; double radius, period; int axis; } orbit; /* axis: 0=x, 1=y, 2=z */
    };
} TrajectoryParams;

Vec3 trajectory_eval(const TrajectoryParams *t, double time);
Vec3 trajectory_velocity(const TrajectoryParams *t, double time);

/* Returns 1 on success, 0 on failure */
int retarded_solve(const TrajectoryParams *traj, Vec3 obs_pos,
                   double t_obs, double *t_emit_out);

#endif
