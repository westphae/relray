#include "scenefile.h"
#include <yaml.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

/* ------------------------------------------------------------------ */
/* Helpers for walking libyaml document nodes                         */
/* ------------------------------------------------------------------ */

static const char *node_str(yaml_document_t *doc, yaml_node_t *node) {
    if (!node || node->type != YAML_SCALAR_NODE) return NULL;
    return (const char *)node->data.scalar.value;
}

static double node_double(yaml_document_t *doc, yaml_node_t *node) {
    const char *s = node_str(doc, node);
    return s ? atof(s) : 0.0;
}

static int node_int(yaml_document_t *doc, yaml_node_t *node) {
    const char *s = node_str(doc, node);
    return s ? atoi(s) : 0;
}

/* Get a child node from a mapping by key name. Returns NULL if not found. */
static yaml_node_t *map_get(yaml_document_t *doc, yaml_node_t *map, const char *key) {
    if (!map || map->type != YAML_MAPPING_NODE) return NULL;
    for (yaml_node_pair_t *p = map->data.mapping.pairs.start;
         p < map->data.mapping.pairs.top; p++) {
        yaml_node_t *k = yaml_document_get_node(doc, p->key);
        if (k && k->type == YAML_SCALAR_NODE &&
            strcmp((const char *)k->data.scalar.value, key) == 0) {
            return yaml_document_get_node(doc, p->value);
        }
    }
    return NULL;
}

/* Read a [x, y, z] sequence into a Vec3. Returns 1 on success. */
static int read_vec3(yaml_document_t *doc, yaml_node_t *node, Vec3 *out) {
    if (!node) return 0;
    if (node->type == YAML_SEQUENCE_NODE) {
        int count = (int)(node->data.sequence.items.top - node->data.sequence.items.start);
        if (count < 3) return 0;
        yaml_node_t *n0 = yaml_document_get_node(doc, node->data.sequence.items.start[0]);
        yaml_node_t *n1 = yaml_document_get_node(doc, node->data.sequence.items.start[1]);
        yaml_node_t *n2 = yaml_document_get_node(doc, node->data.sequence.items.start[2]);
        out->x = node_double(doc, n0);
        out->y = node_double(doc, n1);
        out->z = node_double(doc, n2);
        return 1;
    }
    return 0;
}

/* Count items in a sequence node. Returns 0 if NULL or not a sequence. */
static int seq_len(yaml_node_t *node) {
    if (!node || node->type != YAML_SEQUENCE_NODE) return 0;
    return (int)(node->data.sequence.items.top - node->data.sequence.items.start);
}

/* Get the i-th item of a sequence node. */
static yaml_node_t *seq_get(yaml_document_t *doc, yaml_node_t *node, int i) {
    if (!node || node->type != YAML_SEQUENCE_NODE) return NULL;
    int count = seq_len(node);
    if (i < 0 || i >= count) return NULL;
    return yaml_document_get_node(doc, node->data.sequence.items.start[i]);
}

/* ------------------------------------------------------------------ */
/* SPD parsing                                                        */
/* ------------------------------------------------------------------ */

static int parse_spd(yaml_document_t *doc, yaml_node_t *node, Spd *out) {
    if (!node) return 0;

    /* {rgb: [r, g, b]} */
    yaml_node_t *rgb = map_get(doc, node, "rgb");
    if (rgb) {
        Vec3 c;
        if (!read_vec3(doc, rgb, &c)) {
            fprintf(stderr, "scenefile: rgb requires [r, g, b]\n");
            return 0;
        }
        *out = spd_from_rgb(c.x, c.y, c.z);
        return 1;
    }

    /* {blackbody: {temp, luminance}} */
    yaml_node_t *bb = map_get(doc, node, "blackbody");
    if (bb) {
        yaml_node_t *temp_node = map_get(doc, bb, "temp");
        yaml_node_t *lum_node = map_get(doc, bb, "luminance");
        if (!temp_node || !lum_node) {
            fprintf(stderr, "scenefile: blackbody requires temp and luminance\n");
            return 0;
        }
        double temp = node_double(doc, temp_node);
        double lum = node_double(doc, lum_node);
        *out = spd_blackbody(temp, lum);
        return 1;
    }

    /* {constant: v} */
    yaml_node_t *cst = map_get(doc, node, "constant");
    if (cst) {
        *out = spd_constant(node_double(doc, cst));
        return 1;
    }

    /* {d65: v} */
    yaml_node_t *d65 = map_get(doc, node, "d65");
    if (d65) {
        Spd base = spd_d65();
        *out = spd_scale(&base, node_double(doc, d65));
        return 1;
    }

    /* {monochromatic: {wavelength, power}} */
    yaml_node_t *mono = map_get(doc, node, "monochromatic");
    if (mono) {
        yaml_node_t *wl = map_get(doc, mono, "wavelength");
        yaml_node_t *pw = map_get(doc, mono, "power");
        if (!wl || !pw) {
            fprintf(stderr, "scenefile: monochromatic requires wavelength and power\n");
            return 0;
        }
        *out = spd_monochromatic(node_double(doc, wl), node_double(doc, pw));
        return 1;
    }

    /* {reflectance: [[lambda, value], ...]} */
    yaml_node_t *refl = map_get(doc, node, "reflectance");
    if (refl && refl->type == YAML_SEQUENCE_NODE) {
        int n = seq_len(refl);
        if (n == 0) {
            *out = spd_zero();
            return 1;
        }
        double (*points)[2] = malloc(sizeof(double[2]) * (size_t)n);
        if (!points) return 0;
        for (int i = 0; i < n; i++) {
            yaml_node_t *pair = seq_get(doc, refl, i);
            if (!pair || pair->type != YAML_SEQUENCE_NODE || seq_len(pair) < 2) {
                free(points);
                fprintf(stderr, "scenefile: reflectance entry %d must be [lambda, value]\n", i);
                return 0;
            }
            points[i][0] = node_double(doc, seq_get(doc, pair, 0));
            points[i][1] = node_double(doc, seq_get(doc, pair, 1));
        }
        *out = spd_from_reflectance_curve(points, n);
        free(points);
        return 1;
    }

    fprintf(stderr, "scenefile: unknown SPD type (use rgb, blackbody, constant, d65, monochromatic, or reflectance)\n");
    return 0;
}

/* ------------------------------------------------------------------ */
/* Shape parsing                                                      */
/* ------------------------------------------------------------------ */

static int parse_shape(yaml_document_t *doc, yaml_node_t *node, Shape *out) {
    if (!node || node->type != YAML_MAPPING_NODE) return 0;

    memset(out, 0, sizeof(*out));
    out->rotation = mat3_identity();
    out->inv_rot = mat3_identity();

    int found = 0;

    yaml_node_t *sp = map_get(doc, node, "sphere");
    if (sp) {
        out->type = SHAPE_SPHERE;
        yaml_node_t *r = map_get(doc, sp, "radius");
        out->sphere.radius = r ? node_double(doc, r) : 1.0;
        found = 1;
    }

    yaml_node_t *pl = map_get(doc, node, "plane");
    if (pl) {
        out->type = SHAPE_PLANE;
        found = 1;
    }

    yaml_node_t *bx = map_get(doc, node, "box");
    if (bx) {
        out->type = SHAPE_BOX;
        yaml_node_t *sz = map_get(doc, bx, "size");
        if (sz) read_vec3(doc, sz, &out->box_shape.size);
        found = 1;
    }

    yaml_node_t *cy = map_get(doc, node, "cylinder");
    if (cy) {
        out->type = SHAPE_CYLINDER;
        yaml_node_t *r = map_get(doc, cy, "radius");
        yaml_node_t *h = map_get(doc, cy, "height");
        out->cylinder.radius = r ? node_double(doc, r) : 1.0;
        out->cylinder.height = h ? node_double(doc, h) : 1.0;
        found = 1;
    }

    yaml_node_t *co = map_get(doc, node, "cone");
    if (co) {
        out->type = SHAPE_CONE;
        yaml_node_t *r = map_get(doc, co, "radius");
        yaml_node_t *h = map_get(doc, co, "height");
        out->cone.radius = r ? node_double(doc, r) : 1.0;
        out->cone.height = h ? node_double(doc, h) : 1.0;
        found = 1;
    }

    yaml_node_t *dk = map_get(doc, node, "disk");
    if (dk) {
        out->type = SHAPE_DISK;
        yaml_node_t *r = map_get(doc, dk, "radius");
        out->disk.radius = r ? node_double(doc, r) : 1.0;
        found = 1;
    }

    yaml_node_t *tr = map_get(doc, node, "triangle");
    if (tr) {
        out->type = SHAPE_TRIANGLE;
        yaml_node_t *v0 = map_get(doc, tr, "v0");
        yaml_node_t *v1 = map_get(doc, tr, "v1");
        yaml_node_t *v2 = map_get(doc, tr, "v2");
        if (v0) read_vec3(doc, v0, &out->triangle.v0);
        if (v1) read_vec3(doc, v1, &out->triangle.v1);
        if (v2) read_vec3(doc, v2, &out->triangle.v2);
        found = 1;
    }

    yaml_node_t *py = map_get(doc, node, "pyramid");
    if (py) {
        out->type = SHAPE_PYRAMID;
        yaml_node_t *br = map_get(doc, py, "base_radius");
        yaml_node_t *h = map_get(doc, py, "height");
        yaml_node_t *s = map_get(doc, py, "sides");
        out->pyramid.base_radius = br ? node_double(doc, br) : 1.0;
        out->pyramid.height = h ? node_double(doc, h) : 1.0;
        out->pyramid.sides = s ? node_int(doc, s) : 4;
        found = 1;
    }

    if (!found) {
        fprintf(stderr, "scenefile: no shape type found\n");
        return 0;
    }

    /* Optional position and rotation */
    yaml_node_t *pos = map_get(doc, node, "position");
    yaml_node_t *rot = map_get(doc, node, "rotation");
    if (pos || rot) {
        Vec3 position = VEC3_ZERO;
        Mat3 rotation = mat3_identity();
        if (pos) read_vec3(doc, pos, &position);
        if (rot) {
            Vec3 euler;
            if (read_vec3(doc, rot, &euler)) {
                rotation = mat3_from_euler_deg(euler.x, euler.y, euler.z);
            }
        }
        shape_set_transform(out, position, rotation);
    }

    return 1;
}

/* ------------------------------------------------------------------ */
/* Material parsing                                                   */
/* ------------------------------------------------------------------ */

static int parse_material(yaml_document_t *doc, yaml_node_t *node, Material *out) {
    if (!node || node->type != YAML_MAPPING_NODE) return 0;

    memset(out, 0, sizeof(*out));

    /* {diffuse: <spd>} */
    yaml_node_t *diff = map_get(doc, node, "diffuse");
    if (diff) {
        out->type = MAT_DIFFUSE;
        return parse_spd(doc, diff, &out->diffuse.reflectance);
    }

    /* {mirror: <spd>} */
    yaml_node_t *mirr = map_get(doc, node, "mirror");
    if (mirr) {
        out->type = MAT_MIRROR;
        return parse_spd(doc, mirr, &out->mirror.reflectance);
    }

    /* {glass: {ior, tint: <spd>}} */
    yaml_node_t *glass = map_get(doc, node, "glass");
    if (glass) {
        out->type = MAT_GLASS;
        yaml_node_t *ior_node = map_get(doc, glass, "ior");
        yaml_node_t *tint_node = map_get(doc, glass, "tint");
        out->glass.ior = ior_node ? node_double(doc, ior_node) : 1.5;
        if (tint_node) {
            if (!parse_spd(doc, tint_node, &out->glass.tint)) return 0;
        } else {
            out->glass.tint = spd_constant(1.0);
        }
        return 1;
    }

    /* {checker: {even, odd, scale}} */
    yaml_node_t *chk = map_get(doc, node, "checker");
    if (chk) {
        out->type = MAT_CHECKER;
        yaml_node_t *even = map_get(doc, chk, "even");
        yaml_node_t *odd = map_get(doc, chk, "odd");
        yaml_node_t *scale = map_get(doc, chk, "scale");
        if (!even || !odd) {
            fprintf(stderr, "scenefile: checker requires even and odd\n");
            return 0;
        }
        if (!parse_spd(doc, even, &out->checker.even)) return 0;
        if (!parse_spd(doc, odd, &out->checker.odd)) return 0;
        out->checker.scale = scale ? node_double(doc, scale) : 1.0;
        return 1;
    }

    /* {checker_sphere: {even, odd, num_squares}} */
    yaml_node_t *cs = map_get(doc, node, "checker_sphere");
    if (cs) {
        out->type = MAT_CHECKER_SPHERE;
        yaml_node_t *even = map_get(doc, cs, "even");
        yaml_node_t *odd = map_get(doc, cs, "odd");
        yaml_node_t *nsq = map_get(doc, cs, "num_squares");
        if (!even || !odd) {
            fprintf(stderr, "scenefile: checker_sphere requires even and odd\n");
            return 0;
        }
        if (!parse_spd(doc, even, &out->checker_sphere.even)) return 0;
        if (!parse_spd(doc, odd, &out->checker_sphere.odd)) return 0;
        out->checker_sphere.num_squares = nsq ? node_int(doc, nsq) : 8;
        return 1;
    }

    fprintf(stderr, "scenefile: unknown material type\n");
    return 0;
}

/* ------------------------------------------------------------------ */
/* Trajectory parsing                                                 */
/* ------------------------------------------------------------------ */

static int parse_trajectory(yaml_document_t *doc, yaml_node_t *node, TrajectoryParams *out) {
    if (!node || node->type != YAML_MAPPING_NODE) return 0;

    memset(out, 0, sizeof(*out));

    /* {static: {position: [x,y,z]}} */
    yaml_node_t *st = map_get(doc, node, "static");
    if (st) {
        out->type = TRAJ_STATIC;
        yaml_node_t *pos = map_get(doc, st, "position");
        if (pos) read_vec3(doc, pos, &out->static_traj.position);
        return 1;
    }

    /* {linear: {start: [x,y,z], velocity: [x,y,z]}} */
    yaml_node_t *lin = map_get(doc, node, "linear");
    if (lin) {
        out->type = TRAJ_LINEAR;
        yaml_node_t *start = map_get(doc, lin, "start");
        yaml_node_t *vel = map_get(doc, lin, "velocity");
        if (start) read_vec3(doc, start, &out->linear.start);
        if (vel) read_vec3(doc, vel, &out->linear.velocity);
        double speed = vec3_length(out->linear.velocity);
        if (speed >= SPEED_OF_LIGHT) {
            fprintf(stderr, "scenefile: linear trajectory speed %.3f exceeds c (%.1f)\n",
                    speed, SPEED_OF_LIGHT);
            return 0;
        }
        return 1;
    }

    /* {orbit: {center, radius, period, axis}} */
    yaml_node_t *orb = map_get(doc, node, "orbit");
    if (orb) {
        out->type = TRAJ_ORBIT;
        yaml_node_t *center = map_get(doc, orb, "center");
        yaml_node_t *radius = map_get(doc, orb, "radius");
        yaml_node_t *period = map_get(doc, orb, "period");
        yaml_node_t *axis = map_get(doc, orb, "axis");

        if (center) read_vec3(doc, center, &out->orbit.center);
        out->orbit.radius = radius ? node_double(doc, radius) : 1.0;
        out->orbit.period = period ? node_double(doc, period) : 1.0;

        /* axis: "x"=0, "y"=1 (default), "z"=2 */
        out->orbit.axis = 1; /* y */
        if (axis) {
            const char *a = node_str(doc, axis);
            if (a) {
                if (a[0] == 'x' || a[0] == 'X') out->orbit.axis = 0;
                else if (a[0] == 'z' || a[0] == 'Z') out->orbit.axis = 2;
            }
        }

        /* Validate max orbital speed < c */
        double max_speed = 2.0 * M_PI * out->orbit.radius / out->orbit.period;
        if (max_speed >= SPEED_OF_LIGHT) {
            fprintf(stderr, "scenefile: orbit max speed %.3f exceeds c (%.1f)\n",
                    max_speed, SPEED_OF_LIGHT);
            return 0;
        }
        return 1;
    }

    fprintf(stderr, "scenefile: unknown trajectory type (use static, linear, or orbit)\n");
    return 0;
}

/* ------------------------------------------------------------------ */
/* Sky parsing                                                        */
/* ------------------------------------------------------------------ */

static int parse_sky(yaml_document_t *doc, yaml_node_t *node, SkyParams *out) {
    if (!node || node->type != YAML_MAPPING_NODE) {
        out->type = SKY_NONE;
        return 1;
    }

    yaml_node_t *type_node = map_get(doc, node, "type");
    const char *type_str = node_str(doc, type_node);
    if (!type_str || strcmp(type_str, "none") == 0 || type_str[0] == '\0') {
        out->type = SKY_NONE;
        return 1;
    }

    if (strcmp(type_str, "uniform") == 0) {
        out->type = SKY_UNIFORM;
        yaml_node_t *em = map_get(doc, node, "emission");
        if (!em) {
            fprintf(stderr, "scenefile: uniform sky requires 'emission'\n");
            return 0;
        }
        return parse_spd(doc, em, &out->emission);
    }

    if (strcmp(type_str, "gradient") == 0) {
        out->type = SKY_GRADIENT;
        yaml_node_t *top = map_get(doc, node, "top");
        yaml_node_t *bot = map_get(doc, node, "bottom");
        if (!top || !bot) {
            fprintf(stderr, "scenefile: gradient sky requires 'top' and 'bottom'\n");
            return 0;
        }
        if (!parse_spd(doc, top, &out->top)) return 0;
        if (!parse_spd(doc, bot, &out->bottom)) return 0;
        return 1;
    }

    fprintf(stderr, "scenefile: unknown sky type '%s'\n", type_str);
    return 0;
}

/* ------------------------------------------------------------------ */
/* Top-level scene loading                                            */
/* ------------------------------------------------------------------ */

int scenefile_load(const char *path, SceneFileResult *result) {
    memset(result, 0, sizeof(*result));
    result->scene.sky.type = SKY_NONE;

    FILE *f = fopen(path, "rb");
    if (!f) {
        fprintf(stderr, "scenefile: cannot open '%s'\n", path);
        return 0;
    }

    yaml_parser_t parser;
    yaml_document_t doc;
    int ok = 0;

    if (!yaml_parser_initialize(&parser)) {
        fprintf(stderr, "scenefile: failed to initialize YAML parser\n");
        fclose(f);
        return 0;
    }
    yaml_parser_set_input_file(&parser, f);

    if (!yaml_parser_load(&parser, &doc)) {
        fprintf(stderr, "scenefile: YAML parse error: %s (line %zu)\n",
                parser.problem ? parser.problem : "unknown",
                parser.problem_mark.line + 1);
        yaml_parser_delete(&parser);
        fclose(f);
        return 0;
    }

    yaml_node_t *root = yaml_document_get_root_node(&doc);
    if (!root || root->type != YAML_MAPPING_NODE) {
        fprintf(stderr, "scenefile: root must be a YAML mapping\n");
        goto cleanup;
    }

    /* Name */
    yaml_node_t *name_node = map_get(&doc, root, "name");
    if (name_node) {
        const char *name = node_str(&doc, name_node);
        if (name) {
            strncpy(result->scene.name, name, sizeof(result->scene.name) - 1);
            result->scene.name[sizeof(result->scene.name) - 1] = '\0';
        }
    }

    /* Camera */
    yaml_node_t *cam_node = map_get(&doc, root, "camera");
    if (cam_node && cam_node->type == YAML_MAPPING_NODE) {
        yaml_node_t *pos = map_get(&doc, cam_node, "position");
        yaml_node_t *lat = map_get(&doc, cam_node, "look_at");
        yaml_node_t *up = map_get(&doc, cam_node, "up");
        yaml_node_t *vfov = map_get(&doc, cam_node, "vfov");

        if (pos) read_vec3(&doc, pos, &result->camera.position);
        if (lat) read_vec3(&doc, lat, &result->camera.look_at);
        if (up)  read_vec3(&doc, up, &result->camera.up);
        result->camera.vfov = vfov ? node_double(&doc, vfov) : 60.0;
        result->camera.aspect = 1.0; /* caller overrides */
        result->camera.beta = VEC3_ZERO;
        result->has_camera = 1;
    }

    /* Sky */
    yaml_node_t *sky_node = map_get(&doc, root, "sky");
    if (sky_node) {
        if (!parse_sky(&doc, sky_node, &result->scene.sky)) goto cleanup;
    }

    /* Lights */
    yaml_node_t *lights_node = map_get(&doc, root, "lights");
    if (lights_node && lights_node->type == YAML_SEQUENCE_NODE) {
        int n = seq_len(lights_node);
        result->scene.lights = calloc((size_t)n, sizeof(Light));
        if (!result->scene.lights) goto cleanup;
        for (int i = 0; i < n; i++) {
            yaml_node_t *item = seq_get(&doc, lights_node, i);
            if (!item) continue;
            yaml_node_t *pos = map_get(&doc, item, "position");
            yaml_node_t *em = map_get(&doc, item, "emission");
            Light *l = &result->scene.lights[result->scene.num_lights];
            if (pos) read_vec3(&doc, pos, &l->position);
            if (em) {
                if (!parse_spd(&doc, em, &l->emission)) goto cleanup;
            }
            result->scene.num_lights++;
        }
    }

    /* Objects */
    yaml_node_t *objects_node = map_get(&doc, root, "objects");
    if (objects_node && objects_node->type == YAML_SEQUENCE_NODE) {
        int n = seq_len(objects_node);
        result->scene.objects = calloc((size_t)n, sizeof(Object));
        if (!result->scene.objects) goto cleanup;
        for (int i = 0; i < n; i++) {
            yaml_node_t *item = seq_get(&doc, objects_node, i);
            if (!item) continue;
            yaml_node_t *shape_node = map_get(&doc, item, "shape");
            yaml_node_t *mat_node = map_get(&doc, item, "material");
            Object *obj = &result->scene.objects[result->scene.num_objects];
            if (!parse_shape(&doc, shape_node, &obj->shape)) {
                fprintf(stderr, "scenefile: objects[%d].shape failed\n", i);
                goto cleanup;
            }
            if (!parse_material(&doc, mat_node, &obj->material)) {
                fprintf(stderr, "scenefile: objects[%d].material failed\n", i);
                goto cleanup;
            }
            result->scene.num_objects++;
        }
    }

    /* Moving objects */
    yaml_node_t *moving_node = map_get(&doc, root, "moving_objects");
    if (moving_node && moving_node->type == YAML_SEQUENCE_NODE) {
        int n = seq_len(moving_node);
        result->scene.moving_objects = calloc((size_t)n, sizeof(MovingObject));
        if (!result->scene.moving_objects) goto cleanup;
        for (int i = 0; i < n; i++) {
            yaml_node_t *item = seq_get(&doc, moving_node, i);
            if (!item) continue;
            yaml_node_t *shape_node = map_get(&doc, item, "shape");
            yaml_node_t *mat_node = map_get(&doc, item, "material");
            yaml_node_t *traj_node = map_get(&doc, item, "trajectory");
            MovingObject *mo = &result->scene.moving_objects[result->scene.num_moving];
            if (!parse_shape(&doc, shape_node, &mo->shape)) {
                fprintf(stderr, "scenefile: moving_objects[%d].shape failed\n", i);
                goto cleanup;
            }
            if (!parse_material(&doc, mat_node, &mo->material)) {
                fprintf(stderr, "scenefile: moving_objects[%d].material failed\n", i);
                goto cleanup;
            }
            if (traj_node) {
                if (!parse_trajectory(&doc, traj_node, &mo->trajectory)) {
                    fprintf(stderr, "scenefile: moving_objects[%d].trajectory failed\n", i);
                    goto cleanup;
                }
            }
            result->scene.num_moving++;
        }
    }

    ok = 1;

cleanup:
    yaml_document_delete(&doc);
    yaml_parser_delete(&parser);
    fclose(f);

    if (!ok) {
        scenefile_free(result);
    }
    return ok;
}

void scenefile_free(SceneFileResult *result) {
    free(result->scene.objects);
    result->scene.objects = NULL;
    result->scene.num_objects = 0;

    free(result->scene.moving_objects);
    result->scene.moving_objects = NULL;
    result->scene.num_moving = 0;

    free(result->scene.lights);
    result->scene.lights = NULL;
    result->scene.num_lights = 0;
}
