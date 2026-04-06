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
/* Load with $variable substitution. vars is an array of {name, value} pairs. */
typedef struct { const char *name; double value; } SceneVar;
int scenefile_load_with_vars(const char *path, const SceneVar *vars, int num_vars, SceneFileResult *result);
void scenefile_free(SceneFileResult *result);

#endif
