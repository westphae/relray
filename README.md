# relray

A relativistic ray tracer that renders scenes where the speed of light is reduced to a human scale (~1 m/s). All visual effects — aberration, Doppler shift, searchlight effect, time dilation — emerge naturally from the physics, not as post-processing.

## Building

```
go build ./cmd/relray/
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

All shapes can have optional `position` and `rotation` fields to translate and rotate them.

| Shape | Fields | Use cases |
|-------|--------|-----------|
| `sphere` | `center`, `radius` | Balls, globes, decorations, planets |
| `plane` | `point`, `normal` | Floors, walls, ceilings, mirrors |
| `box` | `min`, `max` | Furniture, buildings, shelves, steps |
| `cylinder` | `radius`, `height` | Table legs, columns, pipes, candles, bottles |
| `cone` | `radius`, `height` | Lamp shades, trees, funnels, rooftops |
| `disk` | `center`, `normal`, `radius` | Table tops, plates, clock faces, mirrors |
| `triangle` | `v0`, `v1`, `v2` | Arbitrary faces, wedges, ramps |
| `pyramid` | `base_radius`, `height`, `sides` | Decorative objects, rooftops, obelisks |

Cylinder, cone, and pyramid are defined in local space (base at Y=0, extending up). Use `position` and `rotation` to place them:

```yaml
- shape:
    cylinder: { radius: 0.05, height: 0.8 }
    position: [1, 0, 2]
    rotation: [0, 0, 90]       # yaw, pitch, roll in degrees
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

Colors and light spectra are defined spectrally, not as RGB. This is essential for physically correct Doppler shifting — wavelengths actually shift, producing realistic color changes.

| Type | Syntax | Description |
|------|--------|-------------|
| RGB | `{ rgb: [r, g, b] }` | Approximate SPD from linear sRGB (for reflectances) |
| Blackbody | `{ blackbody: { temp: 5778, luminance: 1.0 } }` | Planck function at given temperature |
| D65 | `{ d65: 1.0 }` | CIE D65 daylight illuminant, scaled |
| Constant | `{ constant: 0.5 }` | Flat spectrum across all wavelengths |
| Monochromatic | `{ monochromatic: { wavelength: 550, power: 1.0 } }` | Single wavelength (nm) |

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
- **Doppler shift**: Colors shift blue (approaching) or red (receding) for both observer and source motion
- **Searchlight effect**: D³ intensity scaling — forward hemisphere brightens, backward dims
- **Retarded time**: Moving objects appear where they were when the light left them
- **Penrose-Terrell rotation**: A moving sphere still appears spherical (use checker_sphere to see the rotation)

## Example scenes

- `scenes/spheres.yaml` — Test scene with colored, mirror, and glass spheres on a checkerboard
- `scenes/room.yaml` — Living room with furniture, glass globe, mirror sphere, and an orbiting checker globe
- `scenes/shapes_demo.yaml` — Showcase of all shape primitives
