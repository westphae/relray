#ifndef RENDER_H
#define RENDER_H

#include "scene.h"
#include "camera.h"
#include <stdint.h>

typedef struct {
    int width, height;
    int max_depth;
    int samples_per_px;
} RenderConfig;

// Renders a single frame. Returns RGBA pixel buffer (caller must free).
uint8_t *render_frame(const RenderConfig *cfg, const Scene *scene, Camera *cam);

#endif
