#define STB_IMAGE_WRITE_IMPLEMENTATION
#include "stb_image_write.h"
#include "output.h"
#include <stdio.h>
#include <stdlib.h>

int output_save_png(const char *filename, int width, int height, const uint8_t *rgba) {
    return stbi_write_png(filename, width, height, 4, rgba, width * 4);
}

int output_assemble_video(const char *pattern, int fps, const char *out_path) {
    char cmd[1024];
    snprintf(cmd, sizeof(cmd),
        "ffmpeg -y -framerate %d -i \"%s\" -c:v libx264 -pix_fmt yuv420p \"%s\"",
        fps, pattern, out_path);
    return system(cmd);
}
