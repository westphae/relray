#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>
#include <time.h>
#include <math.h>
#include <unistd.h>
#include <sys/stat.h>

#include "vec.h"
#include "spd.h"
#include "shape.h"
#include "material.h"
#include "scene.h"
#include "camera.h"
#include "render.h"
#include "output.h"
#ifdef HAS_CUDA
#include "render_cuda.h"
#endif
#include "retarded.h"
#include "scenefile.h"

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

/* ------------------------------------------------------------------ */
/* Forward declarations                                               */
/* ------------------------------------------------------------------ */

static void print_usage(void);
static void build_spheres_scene(Scene *sc);
static void build_room_scene(Scene *sc);
static Camera camera_preset(const char *scene_name, int width, int height, double beta);
static void run_single(RenderConfig cfg, Scene *sc, int width, int height, double beta, const char *out_file, int use_gpu);
static void run_sweep(RenderConfig cfg, Scene *sc, int width, int height, double beta_min, double beta_max, double beta_step, int fps, const char *out_file, int use_gpu);
static void run_walk(RenderConfig cfg, Scene *sc, int width, int height, double duration, double speed, int fps, const char *out_file, int use_gpu);

/* ------------------------------------------------------------------ */
/* Common option parsing                                              */
/* ------------------------------------------------------------------ */

typedef struct {
    int width, height, samples, depth;
    int use_gpu;
    char scene_name[64];
    char file[512];
    char out[512];
} CommonFlags;

static void common_defaults(CommonFlags *cf) {
    cf->width = 800;
    cf->height = 600;
    cf->samples = 32;
    cf->depth = 8;
    cf->use_gpu = 0;
    strcpy(cf->scene_name, "spheres");
    cf->file[0] = '\0';
    cf->out[0] = '\0';
}

static RenderConfig make_config(const CommonFlags *cf) {
    return (RenderConfig){
        .width = cf->width,
        .height = cf->height,
        .max_depth = cf->depth,
        .samples_per_px = cf->samples,
    };
}

/* Load scene from --file or built-in --scene. */
static void load_scene(const CommonFlags *cf, Scene *sc, Camera *cam, int *has_cam) {
    *has_cam = 0;
    if (cf->file[0] != '\0') {
        SceneFileResult sfr;
        if (!scenefile_load(cf->file, &sfr)) {
            fprintf(stderr, "Error loading scene file: %s\n", cf->file);
            exit(1);
        }
        *sc = sfr.scene;
        if (sfr.has_camera) {
            *cam = sfr.camera;
            *has_cam = 1;
        }
        return;
    }
    if (strcmp(cf->scene_name, "room") == 0) {
        build_room_scene(sc);
    } else {
        build_spheres_scene(sc);
    }
}

/* ------------------------------------------------------------------ */
/* getopt_long option definitions                                     */
/* ------------------------------------------------------------------ */

/* Shared long options (IDs 256+) */
enum {
    OPT_WIDTH = 256, OPT_HEIGHT, OPT_SAMPLES, OPT_DEPTH,
    OPT_SCENE, OPT_FILE, OPT_OUT,
    OPT_BETA, OPT_BETA_MIN, OPT_BETA_MAX, OPT_BETA_STEP,
    OPT_FPS, OPT_DURATION, OPT_SPEED, OPT_GPU,
};

/* "render" subcommand options */
static struct option render_opts[] = {
    {"width",   required_argument, NULL, OPT_WIDTH},
    {"height",  required_argument, NULL, OPT_HEIGHT},
    {"samples", required_argument, NULL, OPT_SAMPLES},
    {"depth",   required_argument, NULL, OPT_DEPTH},
    {"scene",   required_argument, NULL, OPT_SCENE},
    {"file",    required_argument, NULL, OPT_FILE},
    {"out",     required_argument, NULL, OPT_OUT},
    {"beta",    required_argument, NULL, OPT_BETA},
    {"gpu",     no_argument,       NULL, OPT_GPU},
    {"help",    no_argument,       NULL, 'h'},
    {NULL, 0, NULL, 0},
};

/* "sweep" subcommand options */
static struct option sweep_opts[] = {
    {"width",     required_argument, NULL, OPT_WIDTH},
    {"height",    required_argument, NULL, OPT_HEIGHT},
    {"samples",   required_argument, NULL, OPT_SAMPLES},
    {"depth",     required_argument, NULL, OPT_DEPTH},
    {"scene",     required_argument, NULL, OPT_SCENE},
    {"file",      required_argument, NULL, OPT_FILE},
    {"out",       required_argument, NULL, OPT_OUT},
    {"beta-min",  required_argument, NULL, OPT_BETA_MIN},
    {"beta-max",  required_argument, NULL, OPT_BETA_MAX},
    {"beta-step", required_argument, NULL, OPT_BETA_STEP},
    {"fps",       required_argument, NULL, OPT_FPS},
    {"help",      no_argument,       NULL, 'h'},
    {NULL, 0, NULL, 0},
};

/* "walk" subcommand options */
static struct option walk_opts[] = {
    {"width",    required_argument, NULL, OPT_WIDTH},
    {"height",   required_argument, NULL, OPT_HEIGHT},
    {"samples",  required_argument, NULL, OPT_SAMPLES},
    {"depth",    required_argument, NULL, OPT_DEPTH},
    {"scene",    required_argument, NULL, OPT_SCENE},
    {"file",     required_argument, NULL, OPT_FILE},
    {"out",      required_argument, NULL, OPT_OUT},
    {"duration", required_argument, NULL, OPT_DURATION},
    {"speed",    required_argument, NULL, OPT_SPEED},
    {"fps",      required_argument, NULL, OPT_FPS},
    {"help",     no_argument,       NULL, 'h'},
    {NULL, 0, NULL, 0},
};

/* Parse common flags from a getopt result. Returns 1 if consumed. */
static int parse_common(int opt, CommonFlags *cf) {
    switch (opt) {
    case OPT_WIDTH:   cf->width = atoi(optarg); return 1;
    case OPT_HEIGHT:  cf->height = atoi(optarg); return 1;
    case OPT_SAMPLES: cf->samples = atoi(optarg); return 1;
    case OPT_DEPTH:   cf->depth = atoi(optarg); return 1;
    case OPT_SCENE:   strncpy(cf->scene_name, optarg, sizeof(cf->scene_name) - 1); return 1;
    case OPT_FILE:    strncpy(cf->file, optarg, sizeof(cf->file) - 1); return 1;
    case OPT_OUT:     strncpy(cf->out, optarg, sizeof(cf->out) - 1); return 1;
    case OPT_GPU:     cf->use_gpu = 1; return 1;
    default: return 0;
    }
}

/* ------------------------------------------------------------------ */
/* main                                                               */
/* ------------------------------------------------------------------ */

int main(int argc, char **argv) {
    if (argc < 2) {
        print_usage();
        return 0;
    }

    const char *subcmd = argv[1];

    if (strcmp(subcmd, "--help") == 0 || strcmp(subcmd, "-h") == 0 ||
        strcmp(subcmd, "help") == 0) {
        print_usage();
        return 0;
    }

    /* If first arg looks like a flag, treat as "render" */
    if (subcmd[0] == '-') {
        subcmd = "render";
        /* Shift args: insert "render" as argv[1] */
        char **new_argv = malloc(sizeof(char *) * (size_t)(argc + 1));
        new_argv[0] = argv[0];
        new_argv[1] = "render";
        for (int i = 1; i < argc; i++) new_argv[i + 1] = argv[i];
        argc++;
        argv = new_argv;
    }

    /* Reset getopt for subcommand parsing (skip argv[0] and subcmd) */
    optind = 2;

    if (strcmp(subcmd, "render") == 0) {
        CommonFlags cf;
        common_defaults(&cf);
        double beta = 0.0;

        int opt;
        while ((opt = getopt_long(argc, argv, "h", render_opts, NULL)) != -1) {
            if (parse_common(opt, &cf)) continue;
            switch (opt) {
            case OPT_BETA: beta = atof(optarg); break;
            case 'h':
                printf("Usage: crelray render [flags]\n\nRender a single static image.\n\n");
                print_usage();
                return 0;
            default: return 1;
            }
        }

        Scene sc;
        Camera cam;
        int has_cam;
        load_scene(&cf, &sc, &cam, &has_cam);

        if (cf.out[0] == '\0') strcpy(cf.out, "output.png");

        run_single(make_config(&cf), &sc, cf.width, cf.height, beta, cf.out, cf.use_gpu);

    } else if (strcmp(subcmd, "sweep") == 0) {
        CommonFlags cf;
        common_defaults(&cf);
        double beta_min = -0.5, beta_max = 0.5, beta_step = 0.001;
        int fps = 30;

        int opt;
        while ((opt = getopt_long(argc, argv, "h", sweep_opts, NULL)) != -1) {
            if (parse_common(opt, &cf)) continue;
            switch (opt) {
            case OPT_BETA_MIN:  beta_min = atof(optarg); break;
            case OPT_BETA_MAX:  beta_max = atof(optarg); break;
            case OPT_BETA_STEP: beta_step = atof(optarg); break;
            case OPT_FPS:       fps = atoi(optarg); break;
            case 'h':
                printf("Usage: crelray sweep [flags]\n\nRender a beta sweep video.\n\n");
                print_usage();
                return 0;
            default: return 1;
            }
        }

        Scene sc;
        Camera cam;
        int has_cam;
        load_scene(&cf, &sc, &cam, &has_cam);

        if (cf.out[0] == '\0') strcpy(cf.out, "sweep.mp4");

        run_sweep(make_config(&cf), &sc, cf.width, cf.height,
                  beta_min, beta_max, beta_step, fps, cf.out, cf.use_gpu);

    } else if (strcmp(subcmd, "walk") == 0) {
        CommonFlags cf;
        common_defaults(&cf);
        double duration = 10.0, speed = 0.5;
        int fps = 30;

        int opt;
        while ((opt = getopt_long(argc, argv, "h", walk_opts, NULL)) != -1) {
            if (parse_common(opt, &cf)) continue;
            switch (opt) {
            case OPT_DURATION: duration = atof(optarg); break;
            case OPT_SPEED:    speed = atof(optarg); break;
            case OPT_FPS:      fps = atoi(optarg); break;
            case 'h':
                printf("Usage: crelray walk [flags]\n\nRender a walk-through video.\n\n");
                print_usage();
                return 0;
            default: return 1;
            }
        }

        Scene sc;
        Camera cam;
        int has_cam;
        load_scene(&cf, &sc, &cam, &has_cam);

        if (cf.out[0] == '\0') strcpy(cf.out, "walk.mp4");

        run_walk(make_config(&cf), &sc, cf.width, cf.height,
                 duration, speed, fps, cf.out, cf.use_gpu);

    } else {
        fprintf(stderr, "Unknown command: %s\n\n", subcmd);
        print_usage();
        return 1;
    }

    return 0;
}

/* ------------------------------------------------------------------ */
/* Usage                                                              */
/* ------------------------------------------------------------------ */

static void print_usage(void) {
    fprintf(stderr,
        "Usage: crelray <command> [flags]\n"
        "\n"
        "Relativistic ray tracer -- renders scenes with physically correct\n"
        "aberration, Doppler shift, and searchlight effects.\n"
        "\n"
        "Commands:\n"
        "  render    Render a single static image (default)\n"
        "  sweep     Render a beta sweep video across a range of velocities\n"
        "  walk      Render a first-person walk-through video\n"
        "\n"
        "Common flags (all commands):\n"
        "  --width int       image width (default 800)\n"
        "  --height int      image height (default 600)\n"
        "  --samples int     samples per pixel (default 32)\n"
        "  --depth int       max ray bounces (default 8)\n"
        "  --scene string    built-in scene: spheres, room (default \"spheres\")\n"
        "  --file string     load scene from YAML file (overrides --scene)\n"
        "  --out string      output filename\n"
        "  --gpu             use GPU (CUDA) renderer\n"
        "\n"
        "Render flags:\n"
        "  --beta float      observer speed as fraction of c (default 0.0)\n"
        "\n"
        "Sweep flags:\n"
        "  --beta-min float  starting beta (default -0.5)\n"
        "  --beta-max float  ending beta (default 0.5)\n"
        "  --beta-step float beta increment per frame (default 0.001)\n"
        "  --fps int         video framerate (default 30)\n"
        "\n"
        "Walk flags:\n"
        "  --duration float  walk duration in seconds (default 10.0)\n"
        "  --speed float     observer speed as fraction of c (default 0.5)\n"
        "  --fps int         video framerate (default 30)\n"
        "\n"
        "Run 'crelray <command> --help' for command-specific flags.\n"
    );
}

/* ------------------------------------------------------------------ */
/* Camera presets                                                     */
/* ------------------------------------------------------------------ */

static Camera camera_preset(const char *scene_name, int width, int height, double beta) {
    double aspect = (double)width / (double)height;
    Camera cam;
    memset(&cam, 0, sizeof(cam));
    cam.up = vec3(0, 1, 0);
    cam.aspect = aspect;
    cam.beta = vec3(0, 0, beta);

    if (strcmp(scene_name, "room") == 0) {
        cam.position = vec3(0, 1.0, -0.5);
        cam.look_at  = vec3(0, 0.8, 3.0);
        cam.vfov     = 70;
    } else {
        cam.position = vec3(0, 0.5, -3);
        cam.look_at  = vec3(0, 0.3, 0);
        cam.vfov     = 60;
    }
    return cam;
}

/* ------------------------------------------------------------------ */
/* Subcommand implementations                                         */
/* ------------------------------------------------------------------ */

static double elapsed_sec(struct timespec start, struct timespec end) {
    return (double)(end.tv_sec - start.tv_sec) +
           (double)(end.tv_nsec - start.tv_nsec) / 1e9;
}

static void run_single(RenderConfig cfg, Scene *sc, int width, int height,
                       double beta, const char *out_file, int use_gpu) {
    Camera cam = camera_preset(sc->name, width, height, beta);
    camera_init(&cam);

    printf("Rendering %dx%d, beta=%.3f, %d spp, %d bounces%s\n",
           cfg.width, cfg.height, beta, cfg.samples_per_px, cfg.max_depth,
           use_gpu ? " [GPU]" : "");

    struct timespec start, end;
    clock_gettime(CLOCK_MONOTONIC, &start);
    uint8_t *pixels;
#ifdef HAS_CUDA
    if (use_gpu)
        pixels = render_frame_cuda(&cfg, sc, &cam);
    else
#endif
        pixels = render_frame(&cfg, sc, &cam);
    (void)use_gpu;
    clock_gettime(CLOCK_MONOTONIC, &end);

    printf("Rendered in %.3fs\n", elapsed_sec(start, end));

    if (!output_save_png(out_file, width, height, pixels)) {
        fprintf(stderr, "Error saving PNG\n");
    } else {
        printf("Saved to %s\n", out_file);
    }
    free(pixels);
}

static void run_sweep(RenderConfig cfg, Scene *sc, int width, int height,
                      double beta_min, double beta_max, double beta_step,
                      int fps, const char *out_file, int use_gpu) {
    int num_frames = (int)round((beta_max - beta_min) / beta_step) + 1;
    printf("Beta sweep: %.3f to %.3f, step %.4f (%d frames)\n",
           beta_min, beta_max, beta_step, num_frames);
    printf("Rendering %dx%d, %d spp, %d bounces, %d fps\n",
           cfg.width, cfg.height, cfg.samples_per_px, cfg.max_depth, fps);

    char frame_dir[] = "/tmp/crelray-sweep-XXXXXX";
    if (!mkdtemp(frame_dir)) {
        fprintf(stderr, "Failed to create temp directory\n");
        return;
    }

    struct timespec total_start, total_end;
    clock_gettime(CLOCK_MONOTONIC, &total_start);

    for (int i = 0; i < num_frames; i++) {
        double b = beta_min + (double)i * beta_step;
        if (b > beta_max) b = beta_max;

        Camera cam = camera_preset(sc->name, width, height, b);
        camera_init(&cam);

        struct timespec start, end;
        clock_gettime(CLOCK_MONOTONIC, &start);
        uint8_t *pixels;
#ifdef HAS_CUDA
        if (use_gpu) pixels = render_frame_cuda(&cfg, sc, &cam);
        else
#endif
        pixels = render_frame(&cfg, sc, &cam);
        (void)use_gpu;
        clock_gettime(CLOCK_MONOTONIC, &end);

        char frame_path[1024];
        snprintf(frame_path, sizeof(frame_path), "%s/frame_%04d.png", frame_dir, i);
        output_save_png(frame_path, width, height, pixels);
        free(pixels);

        printf("Frame %d/%d  beta=%+.3f  %.3fs\n",
               i + 1, num_frames, b, elapsed_sec(start, end));
    }

    clock_gettime(CLOCK_MONOTONIC, &total_end);
    printf("All frames rendered in %.3fs\n", elapsed_sec(total_start, total_end));
    printf("Assembling video...\n");

    char pattern[1024];
    snprintf(pattern, sizeof(pattern), "%s/frame_%%04d.png", frame_dir);
    if (output_assemble_video(pattern, fps, out_file) != 0) {
        fprintf(stderr, "ffmpeg assembly failed\n");
    } else {
        printf("Saved to %s\n", out_file);
    }

    /* Clean up temp frames */
    for (int i = 0; i < num_frames; i++) {
        char frame_path[1024];
        snprintf(frame_path, sizeof(frame_path), "%s/frame_%04d.png", frame_dir, i);
        remove(frame_path);
    }
    rmdir(frame_dir);
}

static void run_walk(RenderConfig cfg, Scene *sc, int width, int height,
                     double duration, double speed, int fps,
                     const char *out_file, int use_gpu) {
    int num_frames = (int)(duration * (double)fps);
    double dt = 1.0 / (double)fps;
    printf("Walk-through: %.1fs at speed %.2f c, %d frames\n",
           duration, speed, num_frames);
    printf("Rendering %dx%d, %d spp, %d bounces, %d fps\n",
           cfg.width, cfg.height, cfg.samples_per_px, cfg.max_depth, fps);

    char frame_dir[] = "/tmp/crelray-walk-XXXXXX";
    if (!mkdtemp(frame_dir)) {
        fprintf(stderr, "Failed to create temp directory\n");
        return;
    }

    double start_z = -2.0;
    double eye_y = 1.0;
    double aspect = (double)width / (double)height;

    struct timespec total_start, total_end;
    clock_gettime(CLOCK_MONOTONIC, &total_start);

    for (int i = 0; i < num_frames; i++) {
        double t = (double)i * dt;
        double z = start_z + speed * t;

        sc->time = t;

        Camera cam;
        memset(&cam, 0, sizeof(cam));
        cam.position = vec3(0, eye_y, z);
        cam.look_at  = vec3(0, eye_y - 0.1, z + 2);
        cam.up       = vec3(0, 1, 0);
        cam.vfov     = 70;
        cam.aspect   = aspect;
        cam.beta     = vec3(0, 0, speed);
        camera_init(&cam);

        struct timespec start, end;
        clock_gettime(CLOCK_MONOTONIC, &start);
        uint8_t *pixels;
#ifdef HAS_CUDA
        if (use_gpu) pixels = render_frame_cuda(&cfg, sc, &cam);
        else
#endif
        pixels = render_frame(&cfg, sc, &cam);
        (void)use_gpu;
        clock_gettime(CLOCK_MONOTONIC, &end);

        char frame_path[1024];
        snprintf(frame_path, sizeof(frame_path), "%s/frame_%05d.png", frame_dir, i);
        output_save_png(frame_path, width, height, pixels);
        free(pixels);

        printf("Frame %d/%d  t=%.2fs  z=%.2f  %.3fs\n",
               i + 1, num_frames, t, z, elapsed_sec(start, end));
    }

    clock_gettime(CLOCK_MONOTONIC, &total_end);
    printf("All frames rendered in %.3fs\n", elapsed_sec(total_start, total_end));
    printf("Assembling video...\n");

    char pattern[1024];
    snprintf(pattern, sizeof(pattern), "%s/frame_%%05d.png", frame_dir);
    if (output_assemble_video(pattern, fps, out_file) != 0) {
        fprintf(stderr, "ffmpeg assembly failed\n");
    } else {
        printf("Saved to %s\n", out_file);
    }

    /* Clean up temp frames */
    for (int i = 0; i < num_frames; i++) {
        char frame_path[1024];
        snprintf(frame_path, sizeof(frame_path), "%s/frame_%05d.png", frame_dir, i);
        remove(frame_path);
    }
    rmdir(frame_dir);
}

/* ------------------------------------------------------------------ */
/* Built-in scenes                                                    */
/* ------------------------------------------------------------------ */

/* Helper: create a shape positioned at (x, y, z) with no rotation. */
static Shape shape_at(Shape base, double x, double y, double z) {
    shape_set_transform(&base, vec3(x, y, z), mat3_identity());
    return base;
}

/* Helper: create a shape positioned and rotated (Euler degrees). */
static Shape shape_at_rot(Shape base, double x, double y, double z,
                          double yaw, double pitch, double roll) {
    shape_set_transform(&base, vec3(x, y, z), mat3_from_euler_deg(yaw, pitch, roll));
    return base;
}

/* Helper: create a box shape at a given center. */
static Shape box_at(double w, double h, double d, double cx, double cy, double cz) {
    Shape s;
    memset(&s, 0, sizeof(s));
    s.type = SHAPE_BOX;
    s.box_shape.size = vec3(w, h, d);
    s.rotation = mat3_identity();
    s.inv_rot = mat3_identity();
    return shape_at(s, cx, cy, cz);
}

/* Helper: make a sphere shape (no transform). */
static Shape make_sphere(double radius) {
    Shape s;
    memset(&s, 0, sizeof(s));
    s.type = SHAPE_SPHERE;
    s.sphere.radius = radius;
    s.rotation = mat3_identity();
    s.inv_rot = mat3_identity();
    return s;
}

/* Helper: make a plane shape (no transform). */
static Shape make_plane(void) {
    Shape s;
    memset(&s, 0, sizeof(s));
    s.type = SHAPE_PLANE;
    s.rotation = mat3_identity();
    s.inv_rot = mat3_identity();
    return s;
}

static void build_spheres_scene(Scene *sc) {
    memset(sc, 0, sizeof(*sc));
    strncpy(sc->name, "spheres", sizeof(sc->name) - 1);

    Spd sunlight = spd_blackbody(5778, 1.0);
    Spd fill_light = spd_blackbody(7500, 1.0);
    Spd sky_base = spd_blackbody(12000, 1.0);

    /* Sky: gradient */
    sc->sky.type = SKY_GRADIENT;
    sc->sky.top = spd_scale(&sky_base, 0.15);
    sc->sky.bottom = spd_zero();

    /* Lights */
    sc->num_lights = 2;
    sc->lights = calloc(2, sizeof(Light));
    sc->lights[0] = (Light){.position = vec3(2, 5, 0), .emission = spd_scale(&sunlight, 15)};
    sc->lights[1] = (Light){.position = vec3(-3, 3, -2), .emission = spd_scale(&fill_light, 8)};

    /* Objects */
    sc->num_objects = 6;
    sc->objects = calloc(6, sizeof(Object));

    /* Checkerboard floor */
    sc->objects[0] = (Object){
        .shape = shape_at(make_plane(), 0, -0.5, 0),
        .material = {.type = MAT_CHECKER,
                     .checker = {.even = spd_from_rgb(0.7, 0.7, 0.7),
                                 .odd = spd_from_rgb(0.15, 0.15, 0.15),
                                 .scale = 0.5}},
    };

    /* Red sphere */
    sc->objects[1] = (Object){
        .shape = shape_at(make_sphere(0.5), -1.8, 0, 1.5),
        .material = {.type = MAT_DIFFUSE,
                     .diffuse = {.reflectance = spd_from_rgb(0.8, 0.1, 0.1)}},
    };

    /* Green sphere */
    sc->objects[2] = (Object){
        .shape = shape_at(make_sphere(0.5), -0.6, 0, 2),
        .material = {.type = MAT_DIFFUSE,
                     .diffuse = {.reflectance = spd_from_rgb(0.1, 0.8, 0.1)}},
    };

    /* Mirror sphere */
    sc->objects[3] = (Object){
        .shape = shape_at(make_sphere(0.5), 0.6, 0, 2),
        .material = {.type = MAT_MIRROR,
                     .mirror = {.reflectance = spd_constant(0.95)}},
    };

    /* Glass sphere */
    sc->objects[4] = (Object){
        .shape = shape_at(make_sphere(0.5), 1.8, 0, 1.5),
        .material = {.type = MAT_GLASS,
                     .glass = {.ior = 1.5, .tint = spd_constant(1.0)}},
    };

    /* Small blue sphere */
    sc->objects[5] = (Object){
        .shape = shape_at(make_sphere(0.2), 0, -0.3, 1),
        .material = {.type = MAT_DIFFUSE,
                     .diffuse = {.reflectance = spd_from_rgb(0.1, 0.1, 0.8)}},
    };
}

static void build_room_scene(Scene *sc) {
    memset(sc, 0, sizeof(*sc));
    strncpy(sc->name, "room", sizeof(sc->name) - 1);

    Spd sunlight = spd_blackbody(5778, 1.0);
    Spd warm_light = spd_blackbody(3500, 1.0);

    /* Sky: none (indoor) */
    sc->sky.type = SKY_NONE;

    /* Lights */
    sc->num_lights = 3;
    sc->lights = calloc(3, sizeof(Light));
    sc->lights[0] = (Light){.position = vec3(2.5, 2.0, 3.0), .emission = spd_scale(&sunlight, 25)};
    sc->lights[1] = (Light){.position = vec3(0, 2.3, 3.0), .emission = spd_scale(&warm_light, 12)};
    sc->lights[2] = (Light){.position = vec3(2.3, 1.9, 4.9), .emission = spd_scale(&warm_light, 5)};

    /* Materials (reused) */
    Material wall_white = {.type = MAT_DIFFUSE,
                           .diffuse = {.reflectance = spd_from_rgb(0.85, 0.82, 0.78)}};
    Material wall_accent = {.type = MAT_DIFFUSE,
                            .diffuse = {.reflectance = spd_from_rgb(0.6, 0.15, 0.1)}};
    Material glass_mat = {.type = MAT_GLASS,
                          .glass = {.ior = 1.5, .tint = spd_constant(1.0)}};
    Material mirror_mat = {.type = MAT_MIRROR,
                           .mirror = {.reflectance = spd_constant(0.92)}};
    Material floor_wood = {.type = MAT_CHECKER,
                           .checker = {.even = spd_from_rgb(0.55, 0.35, 0.18),
                                       .odd = spd_from_rgb(0.65, 0.45, 0.25),
                                       .scale = 0.4}};
    Material ceiling_mat = {.type = MAT_DIFFUSE,
                            .diffuse = {.reflectance = spd_from_rgb(0.9, 0.9, 0.9)}};
    Material furniture_mat = {.type = MAT_DIFFUSE,
                              .diffuse = {.reflectance = spd_from_rgb(0.3, 0.2, 0.1)}};
    Material cushion_mat = {.type = MAT_DIFFUSE,
                            .diffuse = {.reflectance = spd_from_rgb(0.15, 0.25, 0.5)}};
    Material table_mat = {.type = MAT_DIFFUSE,
                          .diffuse = {.reflectance = spd_from_rgb(0.4, 0.25, 0.12)}};
    Material ball_mat = {.type = MAT_DIFFUSE,
                         .diffuse = {.reflectance = spd_from_rgb(0.9, 0.2, 0.2)}};

    /* Objects: 6 walls + 5 furniture + 3 decorative = 14 */
    sc->num_objects = 14;
    sc->objects = calloc(14, sizeof(Object));
    int idx = 0;

    /* Floor */
    sc->objects[idx++] = (Object){.shape = shape_at(make_plane(), 0, 0, 0), .material = floor_wood};
    /* Ceiling */
    sc->objects[idx++] = (Object){.shape = shape_at_rot(make_plane(), 0, 2.5, 0, 0, 0, 180), .material = ceiling_mat};
    /* Back wall */
    sc->objects[idx++] = (Object){.shape = shape_at_rot(make_plane(), 0, 0, 6, 0, -90, 0), .material = wall_accent};
    /* Left wall */
    sc->objects[idx++] = (Object){.shape = shape_at_rot(make_plane(), -3, 0, 0, 0, 0, 90), .material = wall_white};
    /* Right wall */
    sc->objects[idx++] = (Object){.shape = shape_at_rot(make_plane(), 3, 0, 0, 0, 0, -90), .material = wall_white};
    /* Front wall */
    sc->objects[idx++] = (Object){.shape = shape_at_rot(make_plane(), 0, 0, -2, 0, 90, 0), .material = wall_white};

    /* Coffee table */
    sc->objects[idx++] = (Object){.shape = box_at(1.0, 0.4, 1.0, 0, 0.2, 3.0), .material = table_mat};
    /* Couch base */
    sc->objects[idx++] = (Object){.shape = box_at(1.3, 0.45, 3.0, -2.15, 0.225, 3.0), .material = furniture_mat};
    /* Couch back */
    sc->objects[idx++] = (Object){.shape = box_at(0.3, 0.45, 3.0, -2.65, 0.675, 3.0), .material = furniture_mat};
    /* Couch cushion */
    sc->objects[idx++] = (Object){.shape = box_at(0.9, 0.1, 2.6, -2.05, 0.5, 3.0), .material = cushion_mat};
    /* Bookshelf */
    sc->objects[idx++] = (Object){.shape = box_at(1.0, 1.8, 1.8, 2.3, 0.9, 4.9), .material = furniture_mat};

    /* Glass globe on coffee table */
    sc->objects[idx++] = (Object){.shape = shape_at(make_sphere(0.12), 0.1, 0.55, 3.0), .material = glass_mat};
    /* Mirror sphere on coffee table */
    sc->objects[idx++] = (Object){.shape = shape_at(make_sphere(0.08), -0.2, 0.52, 2.8), .material = mirror_mat};
    /* Red decorative ball */
    sc->objects[idx++] = (Object){.shape = shape_at(make_sphere(0.08), 0.3, 0.5, 3.2), .material = ball_mat};

    /* Moving objects: orbiting checker globe */
    sc->num_moving = 1;
    sc->moving_objects = calloc(1, sizeof(MovingObject));
    sc->moving_objects[0] = (MovingObject){
        .shape = make_sphere(0.12),
        .material = {.type = MAT_CHECKER_SPHERE,
                     .checker_sphere = {.even = spd_from_rgb(0.9, 0.85, 0.15),
                                        .odd = spd_from_rgb(0.1, 0.1, 0.6),
                                        .num_squares = 8}},
        .trajectory = {.type = TRAJ_ORBIT,
                       .orbit = {.center = vec3(0, 1.2, 3.0),
                                 .radius = 0.4,
                                 .period = 10.0,
                                 .axis = 1}}, /* y-axis */
    };
}
