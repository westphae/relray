# relray Implementation Plan

## Context

Build a relativistic ray tracer in Go that renders first-person video of moving through an environment where c ≈ 1 m/s. All relativistic effects emerge from the physics (Lorentz-transformed rays), not post-processing. CPU-only (64 cores), zero external deps.

## Package Structure

```
cmd/relray/main.go          CLI entry, scene definition, render dispatch
pkg/vec/                     Vec3 math
pkg/lorentz/                 Aberration, Doppler, Lorentz boost of null 4-vectors
pkg/spectrum/                SPD type, wavelength shifting, CIE XYZ→sRGB, CIE data tables
pkg/geometry/                Shape interface, Hit struct, sphere, plane
pkg/material/                Material interface, diffuse, mirror, glass
pkg/scene/                   Scene graph (objects + lights), light types
pkg/camera/                  Camera with velocity, rest-frame ray generation
pkg/render/                  Tile-based parallel render loop, ray tracer
pkg/retarded/                Retarded-time solver (Phase 4)
pkg/output/                  PNG writer, FFmpeg video assembly (Phase 4)
```

## Core Render Algorithm

```
For each pixel:
  1. Camera generates ray direction in OBSERVER rest frame
  2. Build null 4-vector k_obs = (1, dx, dy, dz)
  3. Inverse Lorentz boost → k_world (gives aberrated direction AND Doppler factor D)
  4. Trace aberrated ray in world frame → hit point, normal, material
  5. Shade at hit point (diffuse + shadow rays, world frame) → SPD
  6. Apply Doppler: shift SPD wavelengths by factor D, scale intensity by D³
  7. Convert SPD → CIE XYZ → sRGB → pixel
```

This single Lorentz boost naturally produces aberration, Doppler shift, and searchlight effect.

## Parallelism

Tile-based (32×32 pixel tiles) work-stealing via buffered channel. Each goroutine writes to disjoint pixel region — no locks needed. Launch `runtime.NumCPU()` workers.

## Phase 1: Minimal Relativistic Renderer

**Goal**: Single image of static scene (floor plane + 3 colored spheres + point light) from a moving camera showing visible aberration and color shift.

**Implementation order**:
1. `go.mod`, `pkg/vec/` — Vec3 ops + tests
2. `pkg/lorentz/` — Aberrate, DopplerFactor via null 4-vector boost + golden-value tests
3. `pkg/spectrum/`, `cie_data.go` — SPD type, Shift, ToXYZ, XYZToSRGB + round-trip tests
4. `pkg/geometry/` — Shape interface, sphere, plane + intersection tests
5. `pkg/material/` — Material interface, Lambertian diffuse with SPD reflectance
6. `pkg/scene/` — Scene struct, PointLight with SPD emission
7. `pkg/camera/` — Camera with velocity, RayAt
8. `pkg/render/` — Tile-based parallel loop + TraceRay (single-bounce diffuse + shadow)
9. `pkg/output/` — SavePNG
10. `cmd/relray/main.go` — hardcoded scene, render at β=0 (reference) and β=0.5

**Verification**:
- `Aberrate(forward, β=0.5)` → forward (unchanged)
- `Aberrate(sideways, β=0.5)` → tilted to acos(0.5) = 60°
- `DopplerFactor(head-on, β=0.5)` → √3 ≈ 1.732
- `DopplerFactor(receding, β=0.5)` → 1/√3 ≈ 0.577
- Render at β=0 matches standard ray tracer output
- Render at β=0.5 shows forward compression, intensity asymmetry ~27×

## Phase 2: Full Spectral Pipeline

- Proper SPD resampling in `Shift()` with linear interpolation
- Blackbody SPD for light sources (`spectrum/blackbody.go`)
- Measured spectral reflectance curves for materials
- Checkerboard pattern material
- Accumulate SPD through bounces, convert to RGB only at final pixel write

**Verification**: Monochromatic 550nm at β=0.5 head-on → shifted to ~317nm (UV) → renders black

## Phase 3: Complex Materials

- Mirror (specular reflection)
- Glass (Snell's law + Fresnel + spectral dispersion — dispersion is natural with spectral rendering)
- Image-based textures converted to spectral
- Recursive ray tracing for reflection/refraction

## Phase 4: Moving Objects, Scene, Video

- Retarded-time iterative solver (Newton's method on `|P_obs - X_obj(t_emit)|² - c²(t_obs - t_emit)²`)
- Living room scene builder (walls, floor, furniture, window with sunlight)
- Camera path (observer walks through room)
- Frame sequence rendering + FFmpeg video assembly
- Flag parsing: resolution, frame range, velocity profile, output dir

**Verification**:
- Moving sphere perpendicular to LOS: displaced in motion direction by v·d/c
- Moving sphere along LOS: length-contracted to ellipsoid
