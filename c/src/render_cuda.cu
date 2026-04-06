// CUDA-accelerated renderer for crelray
// Uses float32 for SPD operations (64x throughput vs float64 on consumer GPUs).
// Architecture: one thread per sample, accumulate XYZ (3 floats) not SPD (361 floats).

#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <stdint.h>

extern "C" {
#include "vec.h"
#include "spd.h"
#include "shape.h"
#include "material.h"
#include "scene.h"
#include "camera.h"
#include "render.h"
#include "lorentz.h"
#include "retarded.h"
#include "cie_data.h"
}

// ============================================================================
// GPU-side float32 SPD type — reduced to 91 bands (20nm step, 200-2000nm)
// 4x less memory than full 361 bands, negligible quality loss at 8-bit output.
// ============================================================================

#define GPU_BANDS 91
#define GPU_STEP 20.0f
#define GPU_LAMBDA_MIN 200.0f
// Visible range at 20nm step: 380nm = band 9, 780nm = band 29
#define GPU_VIS_START 9
#define GPU_VIS_END 29
// Ratio of GPU step to host step (for downsampling)
/* CPU and GPU both use 91 bands at 20nm step — no downsampling needed. */

typedef struct { float data[GPU_BANDS]; } GSpd;

// Convert CPU SPD (double, 91 bands) to GPU SPD (float, 91 bands).
__host__ __device__ GSpd gspd_from_spd(const Spd *s) {
    GSpd g;
    for (int i = 0; i < GPU_BANDS; i++)
        g.data[i] = (float)s->data[i];
    return g;
}

__device__ GSpd gspd_zero() {
    GSpd s;
    for (int i = 0; i < GPU_BANDS; i++) s.data[i] = 0.0f;
    return s;
}

__device__ void gspd_add_inplace(GSpd *dst, const GSpd *src) {
    for (int i = 0; i < GPU_BANDS; i++) dst->data[i] += src->data[i];
}

__device__ void gspd_mul_inplace(GSpd *dst, const GSpd *src) {
    for (int i = 0; i < GPU_BANDS; i++) dst->data[i] *= src->data[i];
}

__device__ void gspd_scale_inplace(GSpd *dst, float f) {
    for (int i = 0; i < GPU_BANDS; i++) dst->data[i] *= f;
}

__device__ GSpd gspd_shift(const GSpd *s, float factor) {
    GSpd r = gspd_zero();
    if (factor == 1.0f) { for (int i = 0; i < GPU_BANDS; i++) r.data[i] = s->data[i]; return r; }
    float inv = 1.0f / factor;
    float start_idx = (GPU_LAMBDA_MIN * inv - GPU_LAMBDA_MIN) / GPU_STEP;
    float step = inv;
    float idx = start_idx;
    for (int i = 0; i < GPU_BANDS; i++) {
        if (idx >= 0.0f && idx < (float)(GPU_BANDS - 1)) {
            int lo = (int)idx;
            float frac = idx - (float)lo;
            r.data[i] = fmaf(s->data[lo + 1] - s->data[lo], frac, s->data[lo]);
        }
        idx += step;
    }
    return r;
}

// CIE data in constant memory (91 bands, float32, downsampled)
__constant__ float d_cie_x[GPU_BANDS];
__constant__ float d_cie_y[GPU_BANDS];
__constant__ float d_cie_z[GPU_BANDS];

__device__ void gspd_to_xyz(const GSpd *s, float *x, float *y, float *z) {
    *x = *y = *z = 0.0f;
    for (int i = GPU_VIS_START; i <= GPU_VIS_END; i++) {
        *x += s->data[i] * d_cie_x[i];
        *y += s->data[i] * d_cie_y[i];
        *z += s->data[i] * d_cie_z[i];
    }
    // Integration step is GPU_STEP (20nm)
    *x *= GPU_STEP; *y *= GPU_STEP; *z *= GPU_STEP;
}

__device__ float dev_srgb_gamma(float c) {
    return c <= 0.0031308f ? 12.92f * c : 1.055f * powf(c, 1.0f / 2.4f) - 0.055f;
}

__device__ uint8_t dev_to_u8(float v) {
    float c = v * 255.0f;
    return (uint8_t)fminf(fmaxf(c + 0.5f, 0.0f), 255.0f);
}

__device__ void dev_xyz_to_srgb(float x, float y, float z, uint8_t *r, uint8_t *g, uint8_t *b) {
    *r = dev_to_u8(dev_srgb_gamma(3.2406f*x - 1.5372f*y - 0.4986f*z));
    *g = dev_to_u8(dev_srgb_gamma(-0.9689f*x + 1.8758f*y + 0.0415f*z));
    *b = dev_to_u8(dev_srgb_gamma(0.0557f*x - 0.2040f*y + 1.0570f*z));
}

// ============================================================================
// Device-side xoshiro256** RNG
// ============================================================================

struct DevRng { unsigned long long s[4]; };

__device__ unsigned long long dev_rotl(unsigned long long x, int k) {
    return (x << k) | (x >> (64 - k));
}

__device__ unsigned long long dev_rng_next(DevRng *r) {
    unsigned long long result = dev_rotl(r->s[1] * 5, 7) * 9;
    unsigned long long t = r->s[1] << 17;
    r->s[2] ^= r->s[0]; r->s[3] ^= r->s[1];
    r->s[1] ^= r->s[2]; r->s[0] ^= r->s[3];
    r->s[2] ^= t; r->s[3] = dev_rotl(r->s[3], 45);
    return result;
}

__device__ float dev_rng_float(DevRng *r) {
    return (float)(dev_rng_next(r) >> 40) * (1.0f / 16777216.0f);
}

__device__ DevRng dev_rng_seed(unsigned long long seed) {
    DevRng r;
    for (int i = 0; i < 4; i++) {
        seed += 0x9e3779b97f4a7c15ULL;
        unsigned long long z = seed;
        z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9ULL;
        z = (z ^ (z >> 27)) * 0x94d049bb133111ebULL;
        r.s[i] = z ^ (z >> 31);
    }
    return r;
}

// ============================================================================
// Device-side Vec3 (double precision for geometry)
// ============================================================================

__device__ Vec3 dv_add(Vec3 a, Vec3 b) { return {a.x+b.x, a.y+b.y, a.z+b.z}; }
__device__ Vec3 dv_sub(Vec3 a, Vec3 b) { return {a.x-b.x, a.y-b.y, a.z-b.z}; }
__device__ Vec3 dv_scale(Vec3 a, double s) { return {a.x*s, a.y*s, a.z*s}; }
__device__ Vec3 dv_neg(Vec3 a) { return {-a.x, -a.y, -a.z}; }
__device__ double dv_dot(Vec3 a, Vec3 b) { return a.x*b.x + a.y*b.y + a.z*b.z; }
__device__ Vec3 dv_cross(Vec3 a, Vec3 b) {
    return {a.y*b.z-a.z*b.y, a.z*b.x-a.x*b.z, a.x*b.y-a.y*b.x};
}
__device__ double dv_length_sq(Vec3 a) { return dv_dot(a, a); }
__device__ double dv_length(Vec3 a) { return sqrt(dv_length_sq(a)); }
__device__ Vec3 dv_normalize(Vec3 a) {
    double l = dv_length(a);
    return l == 0.0 ? Vec3{0,0,0} : dv_scale(a, 1.0/l);
}
__device__ Vec3 dv_reflect(Vec3 v, Vec3 n) { return dv_sub(v, dv_scale(n, 2.0*dv_dot(v,n))); }
__device__ Vec3 dv_mat3_mul(Mat3 m, Vec3 v) {
    return {dv_dot(m.rows[0],v), dv_dot(m.rows[1],v), dv_dot(m.rows[2],v)};
}

// ============================================================================
// Device-side shape intersection
// ============================================================================

__device__ void dev_hit_set_face_normal(Hit *h, Vec3 ray_dir, Vec3 outward_normal) {
    h->front_face = dv_dot(ray_dir, outward_normal) < 0.0 ? 1 : 0;
    h->normal = h->front_face ? outward_normal : dv_neg(outward_normal);
}

__device__ int dev_intersect_sphere(double radius, Vec3 o, Vec3 d, double tmin, double tmax, Hit *h) {
    double a = dv_length_sq(d), hb = dv_dot(o,d), c = dv_length_sq(o)-radius*radius;
    double disc = hb*hb-a*c;
    if (disc < 0) return 0;
    double sq = sqrt(disc), t = (-hb-sq)/a;
    if (t < tmin || t > tmax) { t = (-hb+sq)/a; if (t < tmin || t > tmax) return 0; }
    h->t = t; h->point = dv_add(o, dv_scale(d, t));
    dev_hit_set_face_normal(h, d, dv_scale(h->point, 1.0/radius)); return 1;
}

__device__ int dev_intersect_plane(Vec3 o, Vec3 d, double tmin, double tmax, Hit *h) {
    if (fabs(d.y) < 1e-12) return 0;
    double t = -o.y/d.y;
    if (t < tmin || t > tmax) return 0;
    h->t = t; h->point = dv_add(o, dv_scale(d, t));
    dev_hit_set_face_normal(h, d, {0,1,0}); return 1;
}

__device__ int dev_intersect_box(Vec3 sz, Vec3 o, Vec3 d, double tmin, double tmax, Hit *h) {
    double hx=sz.x/2,hy=sz.y/2,hz=sz.z/2;
    double ix=1/d.x,iy=1/d.y,iz=1/d.z;
    double t0x=(-hx-o.x)*ix,t1x=(hx-o.x)*ix; if(ix<0){double t=t0x;t0x=t1x;t1x=t;}
    double t0y=(-hy-o.y)*iy,t1y=(hy-o.y)*iy; if(iy<0){double t=t0y;t0y=t1y;t1y=t;}
    double t0z=(-hz-o.z)*iz,t1z=(hz-o.z)*iz; if(iz<0){double t=t0z;t0z=t1z;t1z=t;}
    double tn=fmax(t0x,fmax(t0y,t0z)),tf=fmin(t1x,fmin(t1y,t1z));
    if(tn>tf||tf<tmin||tn>tmax) return 0;
    double t=tn<tmin?tf:tn; if(t>tmax) return 0;
    Vec3 p=dv_add(o,dv_scale(d,t)); Vec3 n={0,0,1}; double b=1e-6;
    if(fabs(p.x+hx)<b) n={-1,0,0}; else if(fabs(p.x-hx)<b) n={1,0,0};
    else if(fabs(p.y+hy)<b) n={0,-1,0}; else if(fabs(p.y-hy)<b) n={0,1,0};
    else if(fabs(p.z+hz)<b) n={0,0,-1};
    h->t=t; h->point=p; dev_hit_set_face_normal(h,d,n); return 1;
}

__device__ int dev_intersect_disk(double r, Vec3 o, Vec3 d, double tmin, double tmax, Hit *h) {
    if(fabs(d.y)<1e-12) return 0;
    double t=-o.y/d.y; if(t<tmin||t>tmax) return 0;
    Vec3 p=dv_add(o,dv_scale(d,t));
    if(p.x*p.x+p.z*p.z>r*r) return 0;
    h->t=t; h->point=p; dev_hit_set_face_normal(h,d,{0,1,0}); return 1;
}

__device__ int dev_intersect_cylinder(double r, double ht, Vec3 o, Vec3 d, double tmin, double tmax, Hit *h) {
    double a=d.x*d.x+d.z*d.z, b=2*(o.x*d.x+o.z*d.z), c=o.x*o.x+o.z*o.z-r*r;
    double bt=-1; Vec3 bn={0,0,0}; int f=0;
    double disc=b*b-4*a*c;
    if(disc>=0&&a>1e-12){double sq=sqrt(disc);
      for(int s=0;s<2;s++){double t=(-b+(s?sq:-sq))/(2*a);
        if(t>=tmin&&t<=tmax){double y=o.y+t*d.y;
          if(y>=0&&y<=ht){bt=t;Vec3 p=dv_add(o,dv_scale(d,t));bn=dv_normalize({p.x,0,p.z});f=1;break;}}}}
    if(fabs(d.y)>1e-12){for(int ci=0;ci<2;ci++){double cap=ci?ht:0;double t=(cap-o.y)/d.y;
      if(t>=tmin&&t<=tmax){Vec3 p=dv_add(o,dv_scale(d,t));
        if(p.x*p.x+p.z*p.z<=r*r&&(!f||t<bt)){bt=t;bn=ci?Vec3{0,1,0}:Vec3{0,-1,0};f=1;}}}}
    if(!f) return 0;
    h->t=bt; h->point=dv_add(o,dv_scale(d,bt)); dev_hit_set_face_normal(h,d,bn); return 1;
}

__device__ int dev_intersect_triangle(Vec3 v0, Vec3 v1, Vec3 v2, Vec3 o, Vec3 d, double tmin, double tmax, Hit *h) {
    Vec3 e1=dv_sub(v1,v0),e2=dv_sub(v2,v0),hv=dv_cross(d,e2);
    double a=dv_dot(e1,hv); if(fabs(a)<1e-12) return 0;
    double f=1/a; Vec3 s=dv_sub(o,v0);
    double u=f*dv_dot(s,hv); if(u<0||u>1) return 0;
    Vec3 q=dv_cross(s,e1); double v=f*dv_dot(d,q); if(v<0||u+v>1) return 0;
    double t=f*dv_dot(e2,q); if(t<tmin||t>tmax) return 0;
    h->t=t; h->point=dv_add(o,dv_scale(d,t));
    dev_hit_set_face_normal(h,d,dv_normalize(dv_cross(e1,e2))); return 1;
}

__device__ int dev_shape_intersect(const Shape *s, Vec3 o, Vec3 d, double tmin, double tmax, Hit *h) {
    Vec3 lo=o,ld=d;
    if(s->has_transform){lo=dv_mat3_mul(s->inv_rot,dv_sub(o,s->position));ld=dv_mat3_mul(s->inv_rot,d);}
    int hit=0;
    switch(s->type){
    case SHAPE_SPHERE:hit=dev_intersect_sphere(s->sphere.radius,lo,ld,tmin,tmax,h);break;
    case SHAPE_PLANE:hit=dev_intersect_plane(lo,ld,tmin,tmax,h);break;
    case SHAPE_BOX:hit=dev_intersect_box(s->box_shape.size,lo,ld,tmin,tmax,h);break;
    case SHAPE_CYLINDER:hit=dev_intersect_cylinder(s->cylinder.radius,s->cylinder.height,lo,ld,tmin,tmax,h);break;
    case SHAPE_DISK:hit=dev_intersect_disk(s->disk.radius,lo,ld,tmin,tmax,h);break;
    case SHAPE_TRIANGLE:hit=dev_intersect_triangle(s->triangle.v0,s->triangle.v1,s->triangle.v2,lo,ld,tmin,tmax,h);break;
    default:break;}
    if(hit&&s->has_transform){h->point=dv_add(dv_mat3_mul(s->rotation,h->point),s->position);h->normal=dv_normalize(dv_mat3_mul(s->rotation,h->normal));}
    return hit;
}

// ============================================================================
// Device-side material scatter
// ============================================================================

__device__ Vec3 dev_random_unit_vec(DevRng *rng) {
    while(1){Vec3 v={dev_rng_float(rng)*2-1,dev_rng_float(rng)*2-1,dev_rng_float(rng)*2-1};
    double l2=dv_length_sq(v);if(l2>1e-6&&l2<=1.0)return dv_scale(v,1.0/sqrt(l2));}
}

typedef struct { MaterialType type; GSpd spd1,spd2; float ior,scale; int num_squares; } GpuMaterial;
typedef struct { int scattered; Vec3 out_dir; GSpd reflectance; } GScatterResult;

__device__ GScatterResult dev_scatter(const GpuMaterial *mat, Vec3 in_dir, const Hit *hit, DevRng *rng) {
    GScatterResult sr; sr.scattered=1; sr.reflectance=gspd_zero(); sr.out_dir={0,0,0};
    switch(mat->type){
    case MAT_DIFFUSE:{Vec3 sc=dv_add(hit->normal,dev_random_unit_vec(rng));
      if(dv_length_sq(sc)<1e-12)sc=hit->normal;
      sr.out_dir=dv_normalize(sc);sr.reflectance=mat->spd1;break;}
    case MAT_MIRROR:sr.out_dir=dv_normalize(dv_reflect(in_dir,hit->normal));sr.reflectance=mat->spd1;break;
    case MAT_GLASS:{double eta=hit->front_face?1.0/mat->ior:mat->ior;
      Vec3 ud=dv_normalize(in_dir);double ci=fmin(-dv_dot(ud,hit->normal),1.0);
      double s2t=eta*eta*(1-ci*ci);double r0=(1-eta)/(1+eta);r0*=r0;
      double refl=r0+(1-r0)*pow(1-ci,5);
      if(s2t>1||dev_rng_float(rng)<refl)sr.out_dir=dv_normalize(dv_reflect(ud,hit->normal));
      else{Vec3 perp=dv_scale(dv_add(ud,dv_scale(hit->normal,ci)),eta);
        sr.out_dir=dv_normalize(dv_add(perp,dv_scale(hit->normal,-sqrt(fabs(1-dv_length_sq(perp))))));}
      sr.reflectance=mat->spd1;break;}
    case MAT_CHECKER:{double inv=1.0/mat->scale;
      Vec3 ref=(fabs(hit->normal.x)>0.9)?Vec3{0,1,0}:Vec3{1,0,0};
      Vec3 t1=dv_normalize(dv_cross(hit->normal,ref)),t2=dv_cross(hit->normal,t1);
      int iu=(int)floor(dv_dot(hit->point,t1)*inv),iv=(int)floor(dv_dot(hit->point,t2)*inv);
      sr.reflectance=((iu+iv)%2==0)?mat->spd1:mat->spd2;
      Vec3 sc=dv_add(hit->normal,dev_random_unit_vec(rng));
      if(dv_length_sq(sc)<1e-12)sc=hit->normal;sr.out_dir=dv_normalize(sc);break;}
    case MAT_CHECKER_SPHERE:{int n=mat->num_squares>0?mat->num_squares:8;
      double lat=asin(fmin(fmax(hit->normal.y,-1.0),1.0)),lon=atan2(hit->normal.z,hit->normal.x);
      int ld=(int)floor(lat*n/M_PI),lnd=(int)floor(lon*n/M_PI);
      sr.reflectance=((ld+lnd)%2==0)?mat->spd1:mat->spd2;
      Vec3 sc=dv_add(hit->normal,dev_random_unit_vec(rng));
      if(dv_length_sq(sc)<1e-12)sc=hit->normal;sr.out_dir=dv_normalize(sc);break;}
    }
    return sr;
}

// ============================================================================
// Moving objects: trajectory evaluation + retarded-time solver
// ============================================================================

typedef struct {
    Shape shape;
    GpuMaterial material;
    TrajectoryParams trajectory;
} GpuMovingObject;

__device__ Vec3 dev_trajectory_eval(const TrajectoryParams *t, double time) {
    switch (t->type) {
    case TRAJ_STATIC: return t->static_traj.position;
    case TRAJ_LINEAR: return dv_add(t->linear.start, dv_scale(t->linear.velocity, time));
    case TRAJ_ORBIT: {
        double angle = 2.0 * M_PI * time / t->orbit.period;
        double c = cos(angle), s = sin(angle);
        Vec3 cen = t->orbit.center; double r = t->orbit.radius;
        switch (t->orbit.axis) {
        case 0: return Vec3{cen.x, cen.y + r*c, cen.z + r*s};
        case 2: return Vec3{cen.x + r*c, cen.y + r*s, cen.z};
        default: return Vec3{cen.x + r*c, cen.y, cen.z + r*s};
        }
    }}
    return Vec3{0,0,0};
}

__device__ Vec3 dev_trajectory_velocity(const TrajectoryParams *t, double time) {
    switch (t->type) {
    case TRAJ_STATIC: return Vec3{0,0,0};
    case TRAJ_LINEAR: return t->linear.velocity;
    case TRAJ_ORBIT: {
        double omega = 2.0 * M_PI / t->orbit.period;
        double angle = omega * time;
        double c = cos(angle), s = sin(angle);
        double rw = t->orbit.radius * omega;
        switch (t->orbit.axis) {
        case 0: return Vec3{0, -rw*s, rw*c};
        case 2: return Vec3{-rw*s, rw*c, 0};
        default: return Vec3{-rw*s, 0, rw*c};
        }
    }}
    return Vec3{0,0,0};
}

__device__ int dev_retarded_solve(const TrajectoryParams *traj, Vec3 obs_pos,
                                   double t_obs, double *t_emit_out) {
    Vec3 pos0 = dev_trajectory_eval(traj, t_obs);
    double dist0 = dv_length(dv_sub(obs_pos, pos0));
    double t_emit = t_obs - dist0; /* c = 1 */

    for (int i = 0; i < 50; i++) {
        Vec3 obj_pos = dev_trajectory_eval(traj, t_emit);
        Vec3 delta = dv_sub(obs_pos, obj_pos);
        double dist = dv_length(delta);
        double td = t_obs - t_emit;
        double f = dist*dist - td*td;
        if (fabs(f) < 1e-10) { *t_emit_out = t_emit; return 1; }
        Vec3 vel = dev_trajectory_velocity(traj, t_emit);
        double fp = -2.0*dv_dot(delta, vel) + 2.0*td;
        if (fabs(fp) < 1e-20) break;
        t_emit -= f / fp;
        if (t_emit > t_obs) t_emit = t_obs - 1e-6;
    }

    Vec3 obj_pos = dev_trajectory_eval(traj, t_emit);
    double dist = dv_length(dv_sub(obs_pos, obj_pos));
    if (fabs(dist - (t_obs - t_emit)) < 1e-6) { *t_emit_out = t_emit; return 1; }
    return 0;
}

// ============================================================================
// Scene intersection (static + moving objects)
// ============================================================================

__device__ int dev_scene_intersect(
    const Shape *shapes, const GpuMaterial *materials, int n,
    const GpuMovingObject *moving, int nm, double scene_time,
    Vec3 o, Vec3 d, double tmin, double tmax,
    Hit *h, const GpuMaterial **out_mat) {
    double cl=tmax; int f=0;

    /* Static objects */
    for(int i=0;i<n;i++){Hit th;th.source_velocity={0,0,0};
      if(dev_shape_intersect(&shapes[i],o,d,tmin,cl,&th)){cl=th.t;*h=th;*out_mat=&materials[i];f=1;}}

    /* Moving objects */
    for(int i=0;i<nm;i++){
      double t_ret=0;
      if(!dev_retarded_solve(&moving[i].trajectory,o,scene_time,&t_ret)) continue;
      Vec3 obj_pos=dev_trajectory_eval(&moving[i].trajectory,t_ret);
      Vec3 obj_vel=dev_trajectory_velocity(&moving[i].trajectory,t_ret);
      Vec3 lo=dv_sub(o,obj_pos);
      Shape ls=moving[i].shape;
      if(ls.has_transform) ls.position={0,0,0};
      Hit th;th.source_velocity={0,0,0};
      if(dev_shape_intersect(&ls,lo,d,tmin,cl,&th)){
        th.point=dv_add(th.point,obj_pos);
        th.source_velocity=obj_vel;
        cl=th.t;*h=th;*out_mat=&moving[i].material;f=1;}}

    return f;
}

__device__ AberrationResult dev_aberrate(Vec3 dir_obs, Vec3 beta) {
    AberrationResult ar; double b2=dv_length_sq(beta);
    if(b2==0){ar.dir=dir_obs;ar.doppler=1;return ar;}
    double g=1/sqrt(1-b2);Vec3 p=dv_neg(dir_obs);double bd=dv_dot(beta,p);
    double kw0=g*(1+bd),fac=(g-1)/b2*bd;
    Vec3 kw=dv_add(p,dv_add(dv_scale(beta,g),dv_scale(beta,fac)));
    ar.dir=dv_normalize(dv_neg(kw));ar.doppler=1/kw0;return ar;
}

typedef struct { int type; GSpd top,bottom,emission; } GpuSky;

__device__ GSpd dev_sky_eval(const GpuSky *sky, Vec3 dir) {
    if(sky->type==2){float t=0.5f*((float)dv_normalize(dir).y+1.0f);
      t=fminf(fmaxf(t,0),1);GSpd r=sky->bottom;gspd_scale_inplace(&r,1-t);
      GSpd tp=sky->top;gspd_scale_inplace(&tp,t);gspd_add_inplace(&r,&tp);return r;}
    if(sky->type==1)return sky->emission;
    return gspd_zero();
}

// ============================================================================
// Iterative path tracer — traces ONE sample, returns XYZ directly
// ============================================================================

__device__ void dev_trace_one_sample(
    const Shape *shapes, const GpuMaterial *materials, int num_objects,
    const GpuMovingObject *moving, int num_moving, double scene_time,
    const Vec3 *light_pos, const GSpd *light_emission, int num_lights,
    const GpuSky *sky,
    Vec3 cam_pos, Vec3 cam_beta, Vec3 cam_u, Vec3 cam_v, Vec3 cam_w,
    double half_w, double half_h,
    int px, int py, int width, int height, int max_depth,
    DevRng *rng, float *out_x, float *out_y, float *out_z)
{
    float inv_w = 1.0f/width, inv_h = 1.0f/height;
    double u = (px + dev_rng_float(rng)) * inv_w;
    double v = 1.0 - (py + dev_rng_float(rng)) * inv_h;
    double sx = (2.0*u-1.0)*half_w, sy = (2.0*v-1.0)*half_h;
    Vec3 dir_obs = dv_normalize(dv_add(dv_add(dv_scale(cam_u, sx), dv_scale(cam_v, sy)), dv_neg(cam_w)));
    AberrationResult ab = dev_aberrate(dir_obs, cam_beta);
    float obs_doppler = (float)ab.doppler;

    // Iterative path trace
    GSpd result = gspd_zero();
    GSpd throughput;
    for (int i = 0; i < NUM_BANDS; i++) throughput.data[i] = 1.0f;
    Vec3 cur_o = cam_pos, cur_d = ab.dir;

    for (int depth = 0; depth < max_depth; depth++) {
        Hit hit; hit.source_velocity = {0,0,0};
        const GpuMaterial *mat = nullptr;
        if (!dev_scene_intersect(shapes, materials, num_objects,
                                  moving, num_moving, scene_time,
                                  cur_o, cur_d, 0.001, 1e12, &hit, &mat)) {
            GSpd sky_spd = dev_sky_eval(sky, cur_d);
            gspd_mul_inplace(&sky_spd, &throughput);
            gspd_add_inplace(&result, &sky_spd);
            break;
        }

        // Direct lighting
        GSpd direct = gspd_zero();
        for (int li = 0; li < num_lights; li++) {
            Vec3 tl = dv_sub(light_pos[li], hit.point);
            double dist = dv_length(tl);
            Vec3 ld = dv_scale(tl, 1.0/dist);
            Hit sh; sh.source_velocity={0,0,0};
            const GpuMaterial *sm;
            if (dev_scene_intersect(shapes, materials, num_objects,
                                     moving, num_moving, scene_time,
                                     hit.point, ld, 0.001, dist-0.001, &sh, &sm)) continue;
            double ct = dv_dot(hit.normal, ld);
            if (ct <= 0) continue;
            float falloff = (float)(ct / (4.0 * M_PI * dist * dist));
            GSpd lc = light_emission[li];
            gspd_scale_inplace(&lc, falloff);
            gspd_add_inplace(&direct, &lc);
        }

        GScatterResult scatter = dev_scatter(mat, cur_d, &hit, rng);

        // Source Doppler for moving objects
        if (dv_length_sq(hit.source_velocity) > 0) {
            Vec3 n_photon = dv_normalize(dv_neg(cur_d));
            Vec3 sv = hit.source_velocity;
            double gamma = 1.0 / sqrt(1.0 - dv_length_sq(sv));
            double d_src = 1.0 / (gamma * (1.0 - dv_dot(sv, n_photon)));
            direct = gspd_shift(&direct, 1.0f / (float)d_src);
            float sd3 = (float)(d_src * d_src * d_src);
            gspd_scale_inplace(&direct, sd3);
        }

        gspd_mul_inplace(&direct, &scatter.reflectance);
        gspd_mul_inplace(&direct, &throughput);
        gspd_add_inplace(&result, &direct);

        if (!scatter.scattered) break;
        gspd_mul_inplace(&throughput, &scatter.reflectance);
        cur_o = hit.point; cur_d = scatter.out_dir;
    }

    // Observer Doppler shift
    result = gspd_shift(&result, 1.0f / obs_doppler);
    float d3 = obs_doppler * obs_doppler * obs_doppler;
    gspd_scale_inplace(&result, d3);

    // Convert to XYZ immediately (3 floats instead of 361)
    gspd_to_xyz(&result, out_x, out_y, out_z);
}

// ============================================================================
// Kernel 1: one thread per sample, accumulates XYZ into shared buffer
// ============================================================================

__global__ void trace_samples_kernel(
    float *xyz_buffer,  // [width * height * 3]
    int width, int height, int spp, int max_depth,
    Vec3 cam_pos, Vec3 cam_beta, Vec3 cam_u, Vec3 cam_v, Vec3 cam_w,
    double half_w, double half_h,
    const Shape *shapes, const GpuMaterial *materials, int num_objects,
    const GpuMovingObject *moving, int num_moving, double scene_time,
    const Vec3 *light_pos, const GSpd *light_emission, int num_lights,
    GpuSky sky)
{
    long long total_samples = (long long)width * height * spp;
    long long tid = (long long)blockIdx.x * blockDim.x + threadIdx.x;
    if (tid >= total_samples) return;

    int pixel_idx = (int)(tid / spp);
    int sample_idx = (int)(tid % spp);
    int px = pixel_idx % width;
    int py = pixel_idx / width;

    DevRng rng = dev_rng_seed((unsigned long long)tid * 31337 + 42 + sample_idx * 7919);

    float sx, sy, sz;
    dev_trace_one_sample(shapes, materials, num_objects,
                         moving, num_moving, scene_time,
                         light_pos, light_emission, num_lights, &sky,
                         cam_pos, cam_beta, cam_u, cam_v, cam_w,
                         half_w, half_h, px, py, width, height, max_depth,
                         &rng, &sx, &sy, &sz);

    // Atomic accumulate XYZ (3 atomics per sample, minimal contention)
    int base = pixel_idx * 3;
    atomicAdd(&xyz_buffer[base + 0], sx);
    atomicAdd(&xyz_buffer[base + 1], sy);
    atomicAdd(&xyz_buffer[base + 2], sz);
}

// ============================================================================
// Kernel 2: convert accumulated XYZ to sRGB pixels
// ============================================================================

__global__ void xyz_to_pixels_kernel(
    uint8_t *pixels, const float *xyz_buffer,
    int width, int height, float inv_spp)
{
    int idx = blockIdx.x * blockDim.x + threadIdx.x;
    if (idx >= width * height) return;

    int base = idx * 3;
    float x = xyz_buffer[base + 0] * inv_spp;
    float y = xyz_buffer[base + 1] * inv_spp;
    float z = xyz_buffer[base + 2] * inv_spp;

    uint8_t r, g, b;
    dev_xyz_to_srgb(x, y, z, &r, &g, &b);
    int pidx = idx * 4;
    pixels[pidx] = r; pixels[pidx+1] = g; pixels[pidx+2] = b; pixels[pidx+3] = 255;
}

// ============================================================================
// Host-side launch
// ============================================================================

extern "C"
uint8_t *render_frame_cuda(const RenderConfig *cfg, const Scene *scene, Camera *cam) {
    camera_init(cam);
    int width = cfg->width, height = cfg->height, npx = width * height;
    int spp = cfg->samples_per_px;

    // Convert CIE data to float32 (already 91 bands, matching GPU_BANDS)
    float cie_xf[GPU_BANDS], cie_yf[GPU_BANDS], cie_zf[GPU_BANDS];
    for (int i = 0; i < GPU_BANDS; i++) {
        cie_xf[i] = (float)cie_x[i];
        cie_yf[i] = (float)cie_y[i];
        cie_zf[i] = (float)cie_z[i];
    }
    cudaMemcpyToSymbol(d_cie_x, cie_xf, sizeof(cie_xf));
    cudaMemcpyToSymbol(d_cie_y, cie_yf, sizeof(cie_yf));
    cudaMemcpyToSymbol(d_cie_z, cie_zf, sizeof(cie_zf));

    // Build GPU arrays
    int no = scene->num_objects, nl = scene->num_lights;
    Shape *h_shapes = (Shape*)malloc(no * sizeof(Shape));
    GpuMaterial *h_mats = (GpuMaterial*)calloc(no, sizeof(GpuMaterial));
    for (int i = 0; i < no; i++) {
        h_shapes[i] = scene->objects[i].shape;
        Material *m = &scene->objects[i].material;
        h_mats[i].type = m->type;
        switch (m->type) {
            case MAT_DIFFUSE: h_mats[i].spd1=gspd_from_spd(&m->diffuse.reflectance); break;
            case MAT_MIRROR:  h_mats[i].spd1=gspd_from_spd(&m->mirror.reflectance); break;
            case MAT_GLASS:   h_mats[i].spd1=gspd_from_spd(&m->glass.tint); h_mats[i].ior=(float)m->glass.ior; break;
            case MAT_CHECKER: h_mats[i].spd1=gspd_from_spd(&m->checker.even); h_mats[i].spd2=gspd_from_spd(&m->checker.odd); h_mats[i].scale=(float)m->checker.scale; break;
            case MAT_CHECKER_SPHERE: h_mats[i].spd1=gspd_from_spd(&m->checker_sphere.even); h_mats[i].spd2=gspd_from_spd(&m->checker_sphere.odd); h_mats[i].num_squares=m->checker_sphere.num_squares; break;
        }
    }
    Vec3 *h_lpos = (Vec3*)malloc(nl * sizeof(Vec3));
    GSpd *h_lemit = (GSpd*)malloc(nl * sizeof(GSpd));
    for (int i = 0; i < nl; i++) { h_lpos[i]=scene->lights[i].position; h_lemit[i]=gspd_from_spd(&scene->lights[i].emission); }

    // Build moving object array
    int nm = scene->num_moving;
    GpuMovingObject *h_moving = (GpuMovingObject*)calloc(nm > 0 ? nm : 1, sizeof(GpuMovingObject));
    for (int i = 0; i < nm; i++) {
        h_moving[i].shape = scene->moving_objects[i].shape;
        h_moving[i].trajectory = scene->moving_objects[i].trajectory;
        Material *m = &scene->moving_objects[i].material;
        h_moving[i].material.type = m->type;
        switch (m->type) {
            case MAT_DIFFUSE: h_moving[i].material.spd1=gspd_from_spd(&m->diffuse.reflectance); break;
            case MAT_MIRROR:  h_moving[i].material.spd1=gspd_from_spd(&m->mirror.reflectance); break;
            case MAT_GLASS:   h_moving[i].material.spd1=gspd_from_spd(&m->glass.tint); h_moving[i].material.ior=(float)m->glass.ior; break;
            case MAT_CHECKER: h_moving[i].material.spd1=gspd_from_spd(&m->checker.even); h_moving[i].material.spd2=gspd_from_spd(&m->checker.odd); h_moving[i].material.scale=(float)m->checker.scale; break;
            case MAT_CHECKER_SPHERE: h_moving[i].material.spd1=gspd_from_spd(&m->checker_sphere.even); h_moving[i].material.spd2=gspd_from_spd(&m->checker_sphere.odd); h_moving[i].material.num_squares=m->checker_sphere.num_squares; break;
        }
    }

    GpuSky gpu_sky; memset(&gpu_sky, 0, sizeof(gpu_sky));
    gpu_sky.type = scene->sky.type;
    gpu_sky.top = gspd_from_spd(&scene->sky.top);
    gpu_sky.bottom = gspd_from_spd(&scene->sky.bottom);
    gpu_sky.emission = gspd_from_spd(&scene->sky.emission);

    // Upload to device
    Shape *d_shapes; GpuMaterial *d_mats; Vec3 *d_lpos; GSpd *d_lemit;
    GpuMovingObject *d_moving;
    uint8_t *d_pixels; float *d_xyz;
    cudaMalloc(&d_shapes, no * sizeof(Shape));
    cudaMalloc(&d_mats, no * sizeof(GpuMaterial));
    cudaMalloc(&d_moving, (nm > 0 ? nm : 1) * sizeof(GpuMovingObject));
    cudaMalloc(&d_lpos, nl * sizeof(Vec3));
    cudaMalloc(&d_lemit, nl * sizeof(GSpd));
    cudaMalloc(&d_pixels, npx * 4);
    cudaMalloc(&d_xyz, npx * 3 * sizeof(float));
    cudaMemset(d_xyz, 0, npx * 3 * sizeof(float));
    cudaMemcpy(d_shapes, h_shapes, no * sizeof(Shape), cudaMemcpyHostToDevice);
    cudaMemcpy(d_mats, h_mats, no * sizeof(GpuMaterial), cudaMemcpyHostToDevice);
    if (nm > 0) cudaMemcpy(d_moving, h_moving, nm * sizeof(GpuMovingObject), cudaMemcpyHostToDevice);
    cudaMemcpy(d_lpos, h_lpos, nl * sizeof(Vec3), cudaMemcpyHostToDevice);
    cudaMemcpy(d_lemit, h_lemit, nl * sizeof(GSpd), cudaMemcpyHostToDevice);

    // Kernel 1: trace all samples (one thread per sample)
    long long total_samples = (long long)npx * spp;
    int block1 = 128;
    int grid1 = (int)((total_samples + block1 - 1) / block1);

    trace_samples_kernel<<<grid1, block1>>>(
        d_xyz, width, height, spp, cfg->max_depth,
        cam->position, cam->velocity, cam->u, cam->v, cam->w, cam->half_w, cam->half_h,
        d_shapes, d_mats, no, d_moving, nm, scene->time,
        d_lpos, d_lemit, nl, gpu_sky);

    cudaError_t err = cudaGetLastError();
    if (err != cudaSuccess) fprintf(stderr, "CUDA launch error: %s\n", cudaGetErrorString(err));

    // Kernel 2: convert XYZ to sRGB
    int block2 = 256;
    int grid2 = (npx + block2 - 1) / block2;
    xyz_to_pixels_kernel<<<grid2, block2>>>(d_pixels, d_xyz, width, height, 1.0f / spp);

    err = cudaDeviceSynchronize();
    if (err != cudaSuccess) fprintf(stderr, "CUDA sync error: %s\n", cudaGetErrorString(err));

    // Download
    uint8_t *pixels = (uint8_t*)calloc(npx * 4, 1);
    cudaMemcpy(pixels, d_pixels, npx * 4, cudaMemcpyDeviceToHost);

    cudaFree(d_shapes); cudaFree(d_mats); cudaFree(d_moving);
    cudaFree(d_lpos); cudaFree(d_lemit);
    cudaFree(d_pixels); cudaFree(d_xyz);
    free(h_shapes); free(h_mats); free(h_moving); free(h_lpos); free(h_lemit);
    return pixels;
}
