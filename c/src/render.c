#include "render.h"
#include "lorentz.h"
#include "rng.h"
#include <math.h>
#include <stdlib.h>
#include <stdatomic.h>
#include <pthread.h>
#include <unistd.h>

#define TILE_SIZE 32

typedef struct { int x0, y0, x1, y1; } Tile;

// Forward declarations
static Spd trace(const Scene *scene, const Camera *cam, int max_depth,
                 double u, double v, Rng *rng);
static Spd trace_world(const Scene *scene, Vec3 origin, Vec3 dir,
                       int depth, Rng *rng);

typedef struct {
    const RenderConfig *cfg;
    const Scene *scene;
    const Camera *cam;
    Tile *tiles;
    int num_tiles;
    atomic_int *next_tile;  // shared across all threads
    uint8_t *pixels;
    int thread_id;
} WorkerCtx;

static void render_tile(const WorkerCtx *ctx, const Tile *tile, Rng *rng) {
    double inv_w = 1.0 / ctx->cfg->width;
    double inv_h = 1.0 / ctx->cfg->height;
    double inv_s = 1.0 / ctx->cfg->samples_per_px;

    for (int y = tile->y0; y < tile->y1; y++) {
        for (int x = tile->x0; x < tile->x1; x++) {
            Spd acc = spd_zero();
            for (int s = 0; s < ctx->cfg->samples_per_px; s++) {
                double jx = rng_float64(rng);
                double jy = rng_float64(rng);
                double u = (x + jx) * inv_w;
                double v = 1.0 - (y + jy) * inv_h;
                Spd sample = trace(ctx->scene, ctx->cam, ctx->cfg->max_depth, u, v, rng);
                spd_add_inplace(&acc, &sample);
            }
            spd_scale_inplace(&acc, inv_s);

            double cx, cy, cz;
            spd_to_xyz(&acc, &cx, &cy, &cz);
            uint8_t r, g, b;
            xyz_to_srgb(cx, cy, cz, &r, &g, &b);

            int idx = (y * ctx->cfg->width + x) * 4;
            ctx->pixels[idx + 0] = r;
            ctx->pixels[idx + 1] = g;
            ctx->pixels[idx + 2] = b;
            ctx->pixels[idx + 3] = 255;
        }
    }
}

static void *worker_func(void *arg) {
    WorkerCtx *ctx = arg;
    Rng rng = rng_seed((uint64_t)ctx->thread_id * 31337);
    while (1) {
        int idx = atomic_fetch_add(ctx->next_tile, 1);
        if (idx >= ctx->num_tiles) break;
        render_tile(ctx, &ctx->tiles[idx], &rng);
    }
    return NULL;
}

uint8_t *render_frame(const RenderConfig *cfg, const Scene *scene, Camera *cam) {
    camera_init(cam);

    int width = cfg->width, height = cfg->height;
    uint8_t *pixels = calloc(width * height * 4, 1);

    // Build tiles
    int nx = (width + TILE_SIZE - 1) / TILE_SIZE;
    int ny = (height + TILE_SIZE - 1) / TILE_SIZE;
    int num_tiles = nx * ny;
    Tile *tiles = malloc(num_tiles * sizeof(Tile));
    int ti = 0;
    for (int y = 0; y < height; y += TILE_SIZE) {
        for (int x = 0; x < width; x += TILE_SIZE) {
            tiles[ti++] = (Tile){
                x, y,
                x + TILE_SIZE < width ? x + TILE_SIZE : width,
                y + TILE_SIZE < height ? y + TILE_SIZE : height
            };
        }
    }

    // Determine thread count
    int num_threads = sysconf(_SC_NPROCESSORS_ONLN);
    if (num_threads < 1) num_threads = 1;

    atomic_int next_tile_counter = 0;

    pthread_t *threads = malloc(num_threads * sizeof(pthread_t));
    WorkerCtx *ctxs = malloc(num_threads * sizeof(WorkerCtx));
    for (int i = 0; i < num_threads; i++) {
        ctxs[i] = (WorkerCtx){
            .cfg = cfg, .scene = scene, .cam = cam,
            .tiles = tiles, .num_tiles = num_tiles,
            .next_tile = &next_tile_counter,
            .pixels = pixels, .thread_id = i,
        };
        pthread_create(&threads[i], NULL, worker_func, &ctxs[i]);
    }
    for (int i = 0; i < num_threads; i++) {
        pthread_join(threads[i], NULL);
    }

    free(threads);
    free(ctxs);
    free(tiles);
    return pixels;
}

// --- Ray tracing ---

static Spd trace(const Scene *scene, const Camera *cam, int max_depth,
                 double u, double v, Rng *rng) {
    Vec3 dir_obs = camera_ray_dir(cam, u, v);
    AberrationResult ab = lorentz_aberrate(dir_obs, cam->beta);

    Spd spd = trace_world(scene, cam->position, ab.dir, max_depth, rng);
    spd = spd_shift(&spd, 1.0 / ab.doppler);
    spd_scale_inplace(&spd, ab.doppler * ab.doppler * ab.doppler);
    return spd;
}

static Spd trace_world(const Scene *scene, Vec3 origin, Vec3 dir,
                       int depth, Rng *rng) {
    if (depth <= 0) return spd_zero();

    Hit hit;
    const Material *mat;
    if (!scene_intersect(scene, origin, dir, 0.001, 1e12, &hit, &mat)) {
        return sky_eval(&scene->sky, dir);
    }

    Spd emitted = material_emitted(mat, &hit);

    // Direct lighting
    Spd direct = spd_zero();
    for (int i = 0; i < scene->num_lights; i++) {
        Vec3 to_light = vec3_sub(scene->lights[i].position, hit.point);
        double dist = vec3_length(to_light);
        Vec3 light_dir = vec3_scale(to_light, 1.0 / dist);

        Hit shadow_hit;
        const Material *shadow_mat;
        if (scene_intersect(scene, hit.point, light_dir, 0.001, dist - 0.001,
                           &shadow_hit, &shadow_mat))
            continue;

        double cos_theta = vec3_dot(hit.normal, light_dir);
        if (cos_theta <= 0) continue;

        double falloff = cos_theta / (4.0 * M_PI * dist * dist);
        Spd light_contrib = scene->lights[i].emission;
        spd_scale_inplace(&light_contrib, falloff);
        spd_add_inplace(&direct, &light_contrib);
    }

    ScatterResult scatter = material_scatter(mat, dir, &hit, rng);
    spd_mul_inplace(&direct, &scatter.reflectance);

    // Indirect lighting
    if (scatter.scattered && depth > 1) {
        Spd bounced = trace_world(scene, hit.point, scatter.out_dir, depth - 1, rng);
        spd_mul_inplace(&bounced, &scatter.reflectance);
        spd_add_inplace(&direct, &bounced);
    }

    // result = emitted + direct
    spd_add_inplace(&emitted, &direct);

    // Source Doppler for moving objects
    if (vec3_length_sq(hit.source_velocity) > 0) {
        Vec3 n_photon = vec3_normalize(vec3_neg(dir));
        Vec3 beta = hit.source_velocity;
        double gamma = 1.0 / sqrt(1.0 - vec3_length_sq(beta));
        double d_source = 1.0 / (gamma * (1.0 - vec3_dot(beta, n_photon)));
        emitted = spd_shift(&emitted, 1.0 / d_source);
        spd_scale_inplace(&emitted, d_source * d_source * d_source);
    }

    return emitted;
}
