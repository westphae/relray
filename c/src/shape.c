#include "shape.h"
#include <math.h>

void hit_set_face_normal(Hit *h, Vec3 ray_dir, Vec3 outward_normal) {
    h->front_face = vec3_dot(ray_dir, outward_normal) < 0;
    h->normal = h->front_face ? outward_normal : vec3_neg(outward_normal);
}

void shape_set_transform(Shape *s, Vec3 position, Mat3 rotation) {
    s->position = position;
    s->rotation = rotation;
    s->inv_rot = mat3_transpose(rotation);
    s->has_transform = 1;
}

/* ---- Per-shape intersection (all origin-centered, local space) ---- */

static int intersect_sphere(double radius, Vec3 o, Vec3 d,
                            double t_min, double t_max, Hit *h) {
    double a = vec3_length_sq(d);
    double half_b = vec3_dot(o, d);
    double c = vec3_length_sq(o) - radius * radius;
    double disc = half_b * half_b - a * c;
    if (disc < 0) return 0;

    double sqrt_disc = sqrt(disc);
    double t = (-half_b - sqrt_disc) / a;
    if (t < t_min || t > t_max) {
        t = (-half_b + sqrt_disc) / a;
        if (t < t_min || t > t_max) return 0;
    }

    Vec3 p = vec3_add(o, vec3_scale(d, t));
    Vec3 outward = vec3_scale(p, 1.0 / radius);
    h->t = t;
    h->point = p;
    hit_set_face_normal(h, d, outward);
    return 1;
}

static int intersect_plane(Vec3 o, Vec3 d,
                           double t_min, double t_max, Hit *h) {
    if (fabs(d.y) < 1e-12) return 0;
    double t = -o.y / d.y;
    if (t < t_min || t > t_max) return 0;
    h->t = t;
    h->point = vec3_add(o, vec3_scale(d, t));
    hit_set_face_normal(h, d, vec3(0, 1, 0));
    return 1;
}

static int intersect_box(Vec3 size, Vec3 o, Vec3 d,
                         double t_min, double t_max, Hit *h) {
    double hx = size.x * 0.5, hy = size.y * 0.5, hz = size.z * 0.5;
    double inv_dx = 1.0 / d.x, inv_dy = 1.0 / d.y, inv_dz = 1.0 / d.z;

    double t0x = (-hx - o.x) * inv_dx;
    double t1x = ( hx - o.x) * inv_dx;
    if (inv_dx < 0) { double tmp = t0x; t0x = t1x; t1x = tmp; }

    double t0y = (-hy - o.y) * inv_dy;
    double t1y = ( hy - o.y) * inv_dy;
    if (inv_dy < 0) { double tmp = t0y; t0y = t1y; t1y = tmp; }

    double t0z = (-hz - o.z) * inv_dz;
    double t1z = ( hz - o.z) * inv_dz;
    if (inv_dz < 0) { double tmp = t0z; t0z = t1z; t1z = tmp; }

    double t_near = fmax(t0x, fmax(t0y, t0z));
    double t_far  = fmin(t1x, fmin(t1y, t1z));

    if (t_near > t_far || t_far < t_min || t_near > t_max)
        return 0;

    double t = t_near;
    if (t < t_min) {
        t = t_far;
        if (t > t_max) return 0;
    }

    Vec3 p = vec3_add(o, vec3_scale(d, t));

    Vec3 normal = {0, 0, 0};
    const double bias = 1e-6;
    if      (fabs(p.x + hx) < bias) normal = vec3(-1, 0, 0);
    else if (fabs(p.x - hx) < bias) normal = vec3( 1, 0, 0);
    else if (fabs(p.y + hy) < bias) normal = vec3(0, -1, 0);
    else if (fabs(p.y - hy) < bias) normal = vec3(0,  1, 0);
    else if (fabs(p.z + hz) < bias) normal = vec3(0, 0, -1);
    else                             normal = vec3(0, 0,  1);

    h->t = t;
    h->point = p;
    hit_set_face_normal(h, d, normal);
    return 1;
}

static int intersect_cylinder(double radius, double height, Vec3 o, Vec3 d,
                              double t_min, double t_max, Hit *h) {
    double a = d.x * d.x + d.z * d.z;
    double b = 2.0 * (o.x * d.x + o.z * d.z);
    double c = o.x * o.x + o.z * o.z - radius * radius;

    double best_t = -1;
    Vec3 best_normal = {0, 0, 0};
    int found = 0;

    double disc = b * b - 4.0 * a * c;
    if (disc >= 0 && a > 1e-12) {
        double sqrt_disc = sqrt(disc);
        double ts[2] = {(-b - sqrt_disc) / (2.0 * a),
                        (-b + sqrt_disc) / (2.0 * a)};
        for (int i = 0; i < 2; i++) {
            double t = ts[i];
            if (t < t_min || t > t_max) continue;
            double y = o.y + t * d.y;
            if (y >= 0 && y <= height) {
                if (!found || t < best_t) {
                    best_t = t;
                    Vec3 p = vec3_add(o, vec3_scale(d, t));
                    best_normal = vec3_normalize(vec3(p.x, 0, p.z));
                    found = 1;
                }
                break; /* first valid from sorted pair is closest */
            }
        }
    }

    /* Test caps */
    if (fabs(d.y) > 1e-12) {
        double caps[2] = {0, height};
        for (int i = 0; i < 2; i++) {
            double t = (caps[i] - o.y) / d.y;
            if (t < t_min || t > t_max) continue;
            Vec3 p = vec3_add(o, vec3_scale(d, t));
            if (p.x * p.x + p.z * p.z <= radius * radius) {
                if (!found || t < best_t) {
                    best_t = t;
                    best_normal = (caps[i] == 0) ? vec3(0, -1, 0) : vec3(0, 1, 0);
                    found = 1;
                }
            }
        }
    }

    if (!found) return 0;
    h->t = best_t;
    h->point = vec3_add(o, vec3_scale(d, best_t));
    hit_set_face_normal(h, d, best_normal);
    return 1;
}

static int intersect_cone(double radius, double height, Vec3 o, Vec3 d,
                          double t_min, double t_max, Hit *h) {
    double k = radius / height;
    double k2 = k * k;
    double hy = height - o.y;

    double a = d.x * d.x + d.z * d.z - k2 * d.y * d.y;
    double b = 2.0 * (o.x * d.x + o.z * d.z) + 2.0 * k2 * hy * d.y;
    double c = o.x * o.x + o.z * o.z - k2 * hy * hy;

    double best_t = 0;
    Vec3 best_normal = {0, 0, 0};
    int found = 0;

    double disc = b * b - 4.0 * a * c;
    if (disc >= 0 && fabs(a) > 1e-12) {
        double sqrt_disc = sqrt(disc);
        double ts[2] = {(-b - sqrt_disc) / (2.0 * a),
                        (-b + sqrt_disc) / (2.0 * a)};
        for (int i = 0; i < 2; i++) {
            double t = ts[i];
            if (t < t_min || t > t_max) continue;
            Vec3 p = vec3_add(o, vec3_scale(d, t));
            if (p.y >= 0 && p.y <= height) {
                if (!found || t < best_t) {
                    best_t = t;
                    double r = sqrt(p.x * p.x + p.z * p.z);
                    if (r > 1e-12)
                        best_normal = vec3_normalize(vec3(p.x / r, k, p.z / r));
                    else
                        best_normal = vec3(0, 1, 0);
                    found = 1;
                }
                break;
            }
        }
    }

    /* Base cap at Y=0 */
    if (fabs(d.y) > 1e-12) {
        double t = -o.y / d.y;
        if (t >= t_min && t <= t_max) {
            Vec3 p = vec3_add(o, vec3_scale(d, t));
            if (p.x * p.x + p.z * p.z <= radius * radius) {
                if (!found || t < best_t) {
                    best_t = t;
                    best_normal = vec3(0, -1, 0);
                    found = 1;
                }
            }
        }
    }

    if (!found) return 0;
    h->t = best_t;
    h->point = vec3_add(o, vec3_scale(d, best_t));
    hit_set_face_normal(h, d, best_normal);
    return 1;
}

static int intersect_disk(double radius, Vec3 o, Vec3 d,
                          double t_min, double t_max, Hit *h) {
    if (fabs(d.y) < 1e-12) return 0;
    double t = -o.y / d.y;
    if (t < t_min || t > t_max) return 0;
    Vec3 p = vec3_add(o, vec3_scale(d, t));
    if (p.x * p.x + p.z * p.z > radius * radius) return 0;
    h->t = t;
    h->point = p;
    hit_set_face_normal(h, d, vec3(0, 1, 0));
    return 1;
}

static int intersect_triangle(Vec3 v0, Vec3 v1, Vec3 v2,
                              Vec3 o, Vec3 d,
                              double t_min, double t_max, Hit *h) {
    Vec3 edge1 = vec3_sub(v1, v0);
    Vec3 edge2 = vec3_sub(v2, v0);
    Vec3 hv = vec3_cross(d, edge2);
    double a = vec3_dot(edge1, hv);

    if (fabs(a) < 1e-12) return 0;

    double f = 1.0 / a;
    Vec3 s = vec3_sub(o, v0);
    double u = f * vec3_dot(s, hv);
    if (u < 0 || u > 1) return 0;

    Vec3 q = vec3_cross(s, edge1);
    double v = f * vec3_dot(d, q);
    if (v < 0 || u + v > 1) return 0;

    double t = f * vec3_dot(edge2, q);
    if (t < t_min || t > t_max) return 0;

    Vec3 p = vec3_add(o, vec3_scale(d, t));
    Vec3 normal = vec3_normalize(vec3_cross(edge1, edge2));
    h->t = t;
    h->point = p;
    hit_set_face_normal(h, d, normal);
    return 1;
}

static int intersect_pyramid(double base_radius, double height, int sides,
                             Vec3 o, Vec3 d,
                             double t_min, double t_max, Hit *h) {
    int n = sides;
    if (n < 3) n = 3;
    if (n > 16) n = 16;

    Vec3 apex = vec3(0, height, 0);
    Vec3 base_center = VEC3_ZERO;

    /* Generate vertices on the fly */
    Vec3 verts[16];
    for (int i = 0; i < n; i++) {
        double angle = 2.0 * M_PI * (double)i / (double)n;
        verts[i] = vec3(base_radius * cos(angle), 0, base_radius * sin(angle));
    }

    Hit closest;
    int found = 0;
    double best = t_max;

    for (int i = 0; i < n; i++) {
        int j = (i + 1) % n;
        Hit tmp;
        /* Side face */
        if (intersect_triangle(apex, verts[i], verts[j], o, d, t_min, best, &tmp)) {
            closest = tmp;
            best = tmp.t;
            found = 1;
        }
        /* Base face */
        if (intersect_triangle(base_center, verts[j], verts[i], o, d, t_min, best, &tmp)) {
            closest = tmp;
            best = tmp.t;
            found = 1;
        }
    }

    if (!found) return 0;
    *h = closest;
    return 1;
}

/* ---- Dispatch ---- */

int shape_intersect(const Shape *s, Vec3 origin, Vec3 dir,
                    double t_min, double t_max, Hit *h) {
    Vec3 o = origin;
    Vec3 d = dir;

    /* Transform ray to local space */
    if (s->has_transform) {
        o = mat3_mul_vec(s->inv_rot, vec3_sub(origin, s->position));
        d = mat3_mul_vec(s->inv_rot, dir);
    }

    int hit_found = 0;
    switch (s->type) {
    case SHAPE_SPHERE:
        hit_found = intersect_sphere(s->sphere.radius, o, d, t_min, t_max, h);
        break;
    case SHAPE_PLANE:
        hit_found = intersect_plane(o, d, t_min, t_max, h);
        break;
    case SHAPE_BOX:
        hit_found = intersect_box(s->box_shape.size, o, d, t_min, t_max, h);
        break;
    case SHAPE_CYLINDER:
        hit_found = intersect_cylinder(s->cylinder.radius, s->cylinder.height,
                                       o, d, t_min, t_max, h);
        break;
    case SHAPE_CONE:
        hit_found = intersect_cone(s->cone.radius, s->cone.height,
                                   o, d, t_min, t_max, h);
        break;
    case SHAPE_DISK:
        hit_found = intersect_disk(s->disk.radius, o, d, t_min, t_max, h);
        break;
    case SHAPE_TRIANGLE:
        hit_found = intersect_triangle(s->triangle.v0, s->triangle.v1,
                                       s->triangle.v2, o, d, t_min, t_max, h);
        break;
    case SHAPE_PYRAMID:
        hit_found = intersect_pyramid(s->pyramid.base_radius, s->pyramid.height,
                                      s->pyramid.sides, o, d, t_min, t_max, h);
        break;
    }

    /* Transform hit back to world space */
    if (hit_found && s->has_transform) {
        h->point = vec3_add(mat3_mul_vec(s->rotation, h->point), s->position);
        h->normal = vec3_normalize(mat3_mul_vec(s->rotation, h->normal));
    }

    return hit_found;
}
