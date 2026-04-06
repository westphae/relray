# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**crelray** is the C/CUDA port of the relray relativistic ray tracer. It renders scenes where the speed of light is ~1 m/s, with all relativistic effects emerging from the physics. Designed for CUDA GPU acceleration using tagged unions (no dynamic dispatch), flat scene arrays, and GPU-compatible PRNG.

See GOAL.md in the Go version for the full project vision, and README.md for usage and scene file format.

## Build

- Build: `make` (auto-detects CUDA via `which nvcc`)
- Build CPU-only: `make HAVE_CUDA=0`
- Install: `make install` or `make install PREFIX=~/.local`
- Clean: `make clean`
- Compiler: GCC with `-O3 -march=native -mavx512f` for AVX-512 auto-vectorization
- CUDA: nvcc with `-arch=sm_89` (Ada Lovelace, RTX 40 series)
- Dependencies: `libyaml`, `stb_image_write.h` (vendored in `include/`)

## Run

```
./crelray render --scene spheres --width 800 --height 600
./crelray render --gpu --file scenes/room.yaml --samples 64
./crelray sweep --scene spheres --beta-min -0.3 --beta-max 0.3
./crelray walk --gpu --scene room --duration 10 --speed 0.3
```

## Source Layout

- `src/vec.h` — Vec3, Mat3 (all inline)
- `src/rng.h` — xoshiro256** PRNG (all inline, GPU-compatible)
- `src/spd.h/c` — 361-band SPD (200-2000nm), CIE color matching, blackbody, from_rgb
- `src/cie_data.h` — CIE 1931 x̄/ȳ/z̄ and D65 arrays (361 elements)
- `src/lorentz.h/c` — Lorentz boost: aberration + Doppler
- `src/retarded.h/c` — Retarded-time solver, parameterized trajectories (no closures)
- `src/shape.h/c` — Tagged union Shape (8 types + transform), intersection functions
- `src/material.h/c` — Tagged union Material (5 types), scatter functions
- `src/scene.h/c` — Scene graph, intersection dispatch with retarded time
- `src/camera.h/c` — Camera with velocity
- `src/render.h/c` — CPU tile-based pthreads renderer with atomic tile counter
- `src/render_cuda.h` — CUDA renderer interface
- `src/render_cuda.cu` — GPU kernel (float32 SPDs, double Vec3, iterative path tracing)
- `src/output.h/c` — PNG output (stb), ffmpeg video assembly
- `src/scenefile.h/c` — YAML scene loader (libyaml)
- `src/main.c` — CLI (getopt_long), built-in scenes, subcommand dispatch

## Architecture Notes

- **No dynamic dispatch**: Shapes and materials are tagged unions with switch dispatch. This is essential for CUDA (no virtual functions on device).
- **Trajectories are parameterized structs**, not closures — C and CUDA can't use function pointers for device code. Trajectory types (static, linear, orbit) are evaluated analytically.
- **GPU path uses float32 SPDs** for 64x throughput vs float64 on consumer GPUs. Geometry remains float64. Visually identical output at 8-bit sRGB.
- **CPU path uses double SPDs** with GCC auto-vectorized AVX-512 loops.
- **Scene files are identical** to the Go and Rust versions — fully interchangeable.
