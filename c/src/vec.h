#ifndef VEC_H
#define VEC_H

#include <math.h>

typedef struct { double x, y, z; } Vec3;

static inline Vec3 vec3(double x, double y, double z) { return (Vec3){x, y, z}; }
static inline Vec3 vec3_add(Vec3 a, Vec3 b) { return (Vec3){a.x+b.x, a.y+b.y, a.z+b.z}; }
static inline Vec3 vec3_sub(Vec3 a, Vec3 b) { return (Vec3){a.x-b.x, a.y-b.y, a.z-b.z}; }
static inline Vec3 vec3_scale(Vec3 a, double s) { return (Vec3){a.x*s, a.y*s, a.z*s}; }
static inline Vec3 vec3_neg(Vec3 a) { return (Vec3){-a.x, -a.y, -a.z}; }
static inline double vec3_dot(Vec3 a, Vec3 b) { return a.x*b.x + a.y*b.y + a.z*b.z; }
static inline Vec3 vec3_cross(Vec3 a, Vec3 b) {
    return (Vec3){a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x};
}
static inline double vec3_length_sq(Vec3 a) { return vec3_dot(a, a); }
static inline double vec3_length(Vec3 a) { return sqrt(vec3_length_sq(a)); }
static inline Vec3 vec3_normalize(Vec3 a) {
    double l = vec3_length(a);
    return l == 0.0 ? (Vec3){0,0,0} : vec3_scale(a, 1.0/l);
}
static inline Vec3 vec3_reflect(Vec3 v, Vec3 n) {
    return vec3_sub(v, vec3_scale(n, 2.0 * vec3_dot(v, n)));
}

// 3x3 matrix stored as 3 row vectors
typedef struct { Vec3 rows[3]; } Mat3;

static inline Mat3 mat3_identity(void) {
    return (Mat3){{{1,0,0}, {0,1,0}, {0,0,1}}};
}

static inline Vec3 mat3_mul_vec(Mat3 m, Vec3 v) {
    return (Vec3){vec3_dot(m.rows[0], v), vec3_dot(m.rows[1], v), vec3_dot(m.rows[2], v)};
}

static inline Mat3 mat3_transpose(Mat3 m) {
    return (Mat3){{{m.rows[0].x, m.rows[1].x, m.rows[2].x},
                   {m.rows[0].y, m.rows[1].y, m.rows[2].y},
                   {m.rows[0].z, m.rows[1].z, m.rows[2].z}}};
}

static inline Mat3 mat3_mul(Mat3 a, Mat3 b) {
    Mat3 bt = mat3_transpose(b);
    Mat3 r;
    for (int i = 0; i < 3; i++)
        r.rows[i] = (Vec3){vec3_dot(a.rows[i], bt.rows[0]),
                            vec3_dot(a.rows[i], bt.rows[1]),
                            vec3_dot(a.rows[i], bt.rows[2])};
    return r;
}

static inline Mat3 mat3_rotation_x(double angle) {
    double c = cos(angle), s = sin(angle);
    return (Mat3){{{1,0,0}, {0,c,-s}, {0,s,c}}};
}

static inline Mat3 mat3_rotation_y(double angle) {
    double c = cos(angle), s = sin(angle);
    return (Mat3){{{c,0,s}, {0,1,0}, {-s,0,c}}};
}

static inline Mat3 mat3_rotation_z(double angle) {
    double c = cos(angle), s = sin(angle);
    return (Mat3){{{c,-s,0}, {s,c,0}, {0,0,1}}};
}

static inline Mat3 mat3_from_euler_deg(double yaw, double pitch, double roll) {
    double y = yaw * M_PI / 180.0;
    double p = pitch * M_PI / 180.0;
    double r = roll * M_PI / 180.0;
    return mat3_mul(mat3_rotation_y(y), mat3_mul(mat3_rotation_x(p), mat3_rotation_z(r)));
}

#define VEC3_ZERO ((Vec3){0, 0, 0})

#endif
