#ifndef RENDER_CUDA_H
#define RENDER_CUDA_H

#ifdef HAS_CUDA

#include "scene.h"
#include "camera.h"
#include "render.h"
#include <stdint.h>

// GPU-accelerated frame rendering. Same interface as CPU render_frame.
uint8_t *render_frame_cuda(const RenderConfig *cfg, const Scene *scene, Camera *cam);

#endif // HAS_CUDA
#endif
