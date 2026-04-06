# crelray

A relativistic ray tracer written in C with optional CUDA GPU acceleration. Renders scenes where the speed of light is reduced to a human scale (~1 m/s). All visual effects — aberration, Doppler shift, searchlight effect, time dilation — emerge naturally from the physics, not as post-processing.

This is the C/CUDA port of [relray](https://github.com/westphae/relray) (Go) and [rrelray](../rrelray) (Rust). It uses the same YAML scene file format and produces identical output. The C version is the fastest CPU implementation (~5.8x faster than the original Go), and the CUDA path enables GPU-accelerated rendering.

## Building

```
make            # builds with CUDA if nvcc is available, CPU-only otherwise
make install    # install to /usr/local/bin (or PREFIX=~/.local make install)
```

Requires: `gcc`, `libyaml`, `make`. Optional: CUDA toolkit (`nvcc`) for GPU support.

On Arch Linux:
```
pacman -S base-devel libyaml
```

## Usage

```
crelray <command> [flags]
```

### Commands

**render** — Render a single static image (default if no command given)
```
crelray render --file scene.yaml --beta 0.3 --width 1600 --height 1200
crelray render --gpu --scene room --beta 0 --out room.png
```

**sweep** — Render a video sweeping observer velocity from beta-min to beta-max
```
crelray sweep --file scene.yaml --beta-min -0.4 --beta-max 0.4 --fps 30
```

**walk** — Render a first-person walk-through video
```
crelray walk --gpu --file scene.yaml --duration 10 --speed 0.3 --fps 30
```

Run `crelray <command> --help` for all flags.

### GPU acceleration

Add `--gpu` to any command to use the CUDA renderer. This runs the ray tracing kernel on the GPU with float32 SPD operations (geometry remains float64). Requires an NVIDIA GPU with compute capability 8.9+ (RTX 40 series).

```
crelray render --gpu --scene spheres --width 1920 --height 1080 --samples 128
```

## Scene files

Scenes are defined in YAML. Use `--file scene.yaml` to load one instead of the built-in scenes (`--scene spheres` or `--scene room`). The scene file format is identical to the Go and Rust versions — scene files are interchangeable.

### Structure

```yaml
name: my_scene

camera:
  position: [0, 1.0, -2]
  look_at: [0, 0.5, 3]
  up: [0, 1, 0]
  vfov: 70

sky:
  type: gradient           # "gradient", "uniform", or "none"
  top: { blackbody: { temp: 12000, luminance: 0.15 } }
  bottom: { constant: 0 }

lights:
  - position: [2, 5, 0]
    emission: { blackbody: { temp: 5778, luminance: 20 } }

objects:
  - shape:
      sphere: { radius: 0.5 }
      position: [0, 0.5, 3]
    material:
      diffuse: { rgb: [0.8, 0.1, 0.1] }

moving_objects:
  - shape: { sphere: { radius: 0.15 } }
    material:
      diffuse: { rgb: [0.2, 0.8, 0.2] }
    trajectory:
      orbit: { center: [0, 1, 3], radius: 0.4, period: 10, axis: y }
```

### Shapes

All shapes are defined at the origin in a canonical orientation and placed in the scene using `position` and `rotation` (Euler angles in degrees: yaw, pitch, roll). Triangle is the only exception — its vertices define its position directly.

| Shape | Intrinsic fields | Default orientation | Use cases |
|-------|-----------------|-------------------|-----------|
| `sphere` | `radius` | centered at origin | Balls, globes, decorations, planets |
| `plane` | (none) | XZ plane, normal +Y | Floors, walls, ceilings |
| `box` | `size: [w, h, d]` | centered at origin | Furniture, buildings, shelves, steps |
| `cylinder` | `radius`, `height` | base at origin, up Y | Table legs, columns, pipes, candles |
| `cone` | `radius`, `height` | base at origin, apex up Y | Lamp shades, trees, funnels |
| `disk` | `radius` | XZ plane, normal +Y | Table tops, plates, clock faces |
| `triangle` | `v0`, `v1`, `v2` | vertices define position | Arbitrary faces, wedges, ramps |
| `pyramid` | `base_radius`, `height`, `sides` | base at origin, up Y | Decorative objects, rooftops |

### Materials

| Material | Fields | Description |
|----------|--------|-------------|
| `diffuse` | SPD | Lambertian (matte) surface |
| `mirror` | SPD | Perfect specular reflection |
| `glass` | `ior`, `tint` (SPD) | Dielectric with refraction and Fresnel reflection |
| `checker` | `even` (SPD), `odd` (SPD), `scale` | Planar checkerboard pattern |
| `checker_sphere` | `even` (SPD), `odd` (SPD), `num_squares` | Spherical checkerboard using lat/lon mapping |

### Spectral power distributions (SPD)

All colors and light spectra are defined spectrally using 361 bands covering **200-2000nm** (near-UV through near-IR) at 5nm resolution. Under Doppler shift, IR energy from blackbody sources correctly blueshifts into the visible range.

| Type | Syntax | Description |
|------|--------|-------------|
| RGB | `{ rgb: [r, g, b] }` | Approximate SPD from linear sRGB, with plausible IR/UV tails |
| Blackbody | `{ blackbody: { temp: 5778, luminance: 1.0 } }` | Planck function across full 200-2000nm range |
| D65 | `{ d65: 1.0 }` | CIE D65 daylight illuminant, extended into UV/IR |
| Constant | `{ constant: 0.5 }` | Flat spectrum across all bands |
| Monochromatic | `{ monochromatic: { wavelength: 550, power: 1.0 } }` | Single wavelength (nm) |
| Reflectance | `{ reflectance: [[λ, v], ...] }` | Measured reflectance curve, linearly interpolated |

### Trajectories (for moving objects)

Moving objects are rendered at their retarded-time position and exhibit source Doppler shift. Trajectories are validated to not exceed the speed of light.

| Type | Fields | Description |
|------|--------|-------------|
| `static` | `position` | Stationary (for testing) |
| `linear` | `start`, `velocity` | Constant velocity, straight line |
| `orbit` | `center`, `radius`, `period`, `axis` | Circular orbit around x/y/z axis |

### Sky

| Type | Fields | Description |
|------|--------|-------------|
| `none` | — | Black (indoor scenes) |
| `uniform` | `emission` (SPD) | Same color in every direction |
| `gradient` | `top` (SPD), `bottom` (SPD) | Linear blend by ray elevation |

## Architecture

Designed for CUDA portability:
- **Tagged unions** for shapes and materials (no function pointers/virtual dispatch)
- **Flat scene arrays** suitable for GPU memory upload
- **xoshiro256\*\*** PRNG (GPU-compatible, no heap allocation)
- **pthreads** with atomic tile counter for CPU parallelism
- **GCC auto-vectorization** with `-O3 -march=native -mavx512f` for SIMD SPD operations

## Performance

Benchmark: spheres scene, 800×600, 64 samples per pixel, 8 bounces.

| Version | Time |
|---------|------|
| Go (original) | 4.66s |
| Go + SIMD (vek) | 1.27s |
| Rust | 1.15s |
| **C (CPU)** | **0.81s** |
| C (GPU, CUDA) | 3.49s (unoptimized, memory-bound) |

## Example scenes

- `scenes/spheres.yaml` — Test scene with colored, mirror, and glass spheres on a checkerboard
- `scenes/room.yaml` — Living room with furniture, glass globe, mirror sphere, and an orbiting checker globe
- `scenes/shapes_demo.yaml` — Showcase of all shape primitives
