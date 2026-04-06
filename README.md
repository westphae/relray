# relray

A relativistic ray tracer that renders scenes where the speed of light is reduced to a human scale (~1 m/s). All visual effects — aberration, Doppler shift, searchlight effect, time dilation — emerge naturally from the physics, not as post-processing.

## Repository structure

This is a multi-language project. The Go implementation lives at the repo root, with Rust and C/CUDA in subdirectories. All share the same `scenes/` directory and YAML scene file format.

```
relray/
  cmd/relray/         Go CLI entry point
  pkg/                Go packages (spectrum, geometry, material, render, ...)
  scenes/             Shared YAML scene files
  rust/               Rust implementation (Cargo)
  c/                  C/CUDA implementation (Make)
```

## Building

**Go** (from repo root):
```
go build ./cmd/relray/
go install ./cmd/relray/
```

**Rust** (from `rust/`):
```
cd rust
cargo build --release
cargo install --path .
```

**C/CUDA** (from `c/`):
```
cd c
make                    # auto-detects CUDA
make install PREFIX=~/.local
```

## Usage

```
relray <command> [flags]
```

### Commands

**render** — Render a single static image (default if no command given)
```
relray render --file scene.yaml --beta 0.3 --width 1600 --height 1200
relray render --scene room --beta 0 --out room.png
```

**sweep** — Render a video sweeping observer velocity from beta-min to beta-max
```
relray sweep --file scene.yaml --beta-min -0.4 --beta-max 0.4 --fps 30
```

**walk** — Render a first-person walk-through video
```
relray walk --file scene.yaml --duration 10 --speed 0.3 --fps 30
```

Run `relray <command> --help` for all flags.

## Scene files

Scenes are defined in YAML. Use `--file scene.yaml` to load one instead of the built-in scenes (`--scene spheres` or `--scene room`).

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
  - shape: { sphere: { center: [0, 0.5, 3], radius: 0.5 } }
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

Examples:

```yaml
# Sphere placed at a specific location
- shape:
    sphere: { radius: 0.5 }
    position: [1, 0, 2]

# Wall: plane rotated so normal points +Z, positioned at Z=-2
- shape:
    plane: {}
    position: [0, 0, -2]
    rotation: [0, 90, 0]

# Tilted cylinder
- shape:
    cylinder: { radius: 0.05, height: 0.8 }
    position: [1, 0, 2]
    rotation: [0, 0, 90]       # yaw, pitch, roll in degrees

# Box positioned by its center
- shape:
    box: { size: [1.0, 0.4, 1.0] }
    position: [0, 0.2, 3.0]
```

### Materials

| Material | Fields | Description |
|----------|--------|-------------|
| `diffuse` | SPD | Lambertian (matte) surface |
| `mirror` | SPD | Perfect specular reflection |
| `glass` | `ior`, `tint` (SPD) | Dielectric with refraction and Fresnel reflection |
| `checker` | `even` (SPD), `odd` (SPD), `scale` | Planar checkerboard pattern |
| `checker_sphere` | `even` (SPD), `odd` (SPD), `num_squares` | Spherical checkerboard using lat/lon mapping |

### Spectral power distributions (SPD)

All colors and light spectra are defined spectrally using 361 bands covering **200-2000nm** (near-UV through near-IR) at 5nm resolution. This extended range is essential for physically correct Doppler shifting — under blueshift, infrared energy from blackbody light sources shifts into the visible range, and under redshift, visible light shifts to IR while UV enters the visible blue end.

CIE color matching integrates only over the visible sub-range (380-780nm). The extra UV/IR bands carry energy that becomes visible when Doppler-shifted.

| Type | Syntax | Description |
|------|--------|-------------|
| RGB | `{ rgb: [r, g, b] }` | Approximate SPD from linear sRGB, with plausible IR/UV tails |
| Blackbody | `{ blackbody: { temp: 5778, luminance: 1.0 } }` | Planck function across full 200-2000nm range |
| D65 | `{ d65: 1.0 }` | CIE D65 daylight illuminant, extended into UV/IR |
| Constant | `{ constant: 0.5 }` | Flat spectrum across all bands |
| Monochromatic | `{ monochromatic: { wavelength: 550, power: 1.0 } }` | Single wavelength (nm) |
| Reflectance | `{ reflectance: [[λ, v], ...] }` | Measured reflectance curve, linearly interpolated |

The `reflectance` type allows physically accurate material definitions with wavelength-dependent reflectance across the full spectral range:

```yaml
material:
  diffuse:
    reflectance:
      - [200, 0.02]    # low UV reflectance
      - [400, 0.05]    # absorbs blue
      - [550, 0.10]    # absorbs green
      - [650, 0.85]    # reflects red
      - [780, 0.80]    # high near-IR reflectance
      - [1200, 0.75]
      - [2000, 0.60]
```

### Trajectories (for moving objects)

Moving objects are rendered at their retarded-time position (where they were when the light left them) and exhibit source Doppler shift. Trajectories are validated to not exceed the speed of light.

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

## Relativistic effects

The renderer produces these effects natively from the physics:

- **Aberration**: Forward compression of the field of view at high speed
- **Doppler shift**: Colors shift blue (approaching) or red (receding) for both observer and source motion. The extended 200-2000nm spectral range means IR light from blackbody sources correctly blueshifts into the visible range, keeping scenes bright at high velocities
- **Searchlight effect**: D³ intensity scaling — forward hemisphere brightens, backward dims
- **Retarded time**: Moving objects appear where they were when the light left them
- **Penrose-Terrell rotation**: A moving sphere still appears spherical (use checker_sphere to see the rotation)
- **Source Doppler**: Moving objects exhibit their own Doppler shift independent of the observer's motion

## Performance

This project has been implemented in Go, Rust, and C/CUDA. All versions share the same YAML scene file format and produce identical output.

Benchmark: room scene, 4000×3000, 256 samples per pixel, 8 bounces (sequential, exclusive CPU access):

| Implementation | Time | vs Go | Notes |
|---------------|------|-------|-------|
| Go (root) | 36m39s | 1.0x | vek SIMD for SPD ops |
| Rust (`rust/`) | 16m14s | 2.3x | LLVM auto-vectorization + LTO |
| C (`c/`) CPU | 16m54s | 2.2x | GCC `-O3 -march=native -mavx512f -flto` |
| C (`c/`) GPU | 3m54s | 9.4x | CUDA, float32 SPD, RTX 4070 Ti SUPER |

All versions use 91 spectral bands (20nm step, 200-2000nm).

## Example scenes

- `scenes/spheres.yaml` — Test scene with colored, mirror, and glass spheres on a checkerboard
- `scenes/room.yaml` — Living room with furniture, glass globe, mirror sphere, and an orbiting checker globe
- `scenes/shapes_demo.yaml` — Showcase of all shape primitives
