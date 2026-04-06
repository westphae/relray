#ifndef OUTPUT_H
#define OUTPUT_H

#include <stdint.h>

int output_save_png(const char *filename, int width, int height, const uint8_t *rgba);
int output_assemble_video(const char *pattern, int fps, const char *out_path);

#endif
