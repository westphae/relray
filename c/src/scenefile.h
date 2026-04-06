#ifndef SCENEFILE_H
#define SCENEFILE_H

#include "scene.h"
#include "camera.h"

typedef struct {
    Scene scene;
    Camera camera;
    int has_camera;
} SceneFileResult;

int scenefile_load(const char *path, SceneFileResult *result);
void scenefile_free(SceneFileResult *result);

#endif
