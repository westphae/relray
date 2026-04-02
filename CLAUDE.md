# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**relray** is a relativistic ray tracer that renders video showing what a human would see moving through a familiar environment where the speed of light is reduced to a human scale (~1m/s). All relativistic visual effects (length contraction, time dilation, aberration, Doppler shift, searchlight effect) emerge naturally from the ray tracing math — nothing is added as post-processing.

See GOAL.md for the full project vision and README.md for usage and scene file format.

## Language & Build

- Language: Go
- Build: `go build ./...`
- Install: `go install ./cmd/relray/` — **run this after every change to cmd/relray or its dependencies**
- Test: `go test ./...`
- Single test: `go test ./pkg/... -run TestName`
- Run: `relray render --file scenes/spheres.yaml`
- Target hardware: 64-core CPU, 256GB RAM, NVIDIA RTX 4070 Ti SUPER (16GB VRAM)
- GPU is not used; the Go implementation is CPU-only, parallelized via tile-based goroutine pool

## CLI

Three subcommands: `render` (single image), `sweep` (beta sweep video), `walk` (walk-through video). Use `--file scene.yaml` to load a YAML scene or `--scene spheres|room` for built-ins. Run `relray --help` or `relray <cmd> --help` for flags.

## Package Structure

- `cmd/relray/` — CLI entry point, built-in scene definitions, subcommand dispatch
- `pkg/vec/` — Vec3, Mat3 (rotation matrices), vector/matrix math
- `pkg/lorentz/` — Lorentz boost of null 4-vectors: aberration + Doppler factor
- `pkg/spectrum/` — 81-band SPD (380-780nm), CIE 1931 color matching, blackbody, sRGB conversion
- `pkg/geometry/` — Shape interface + primitives (sphere, plane, box, cylinder, cone, disk, triangle, pyramid) + transform wrapper
- `pkg/material/` — Material interface + implementations (diffuse, mirror, glass, checker, checker_sphere)
- `pkg/scene/` — Scene graph (static + moving objects, lights, sky)
- `pkg/camera/` — Camera with velocity (beta), rest-frame ray generation
- `pkg/render/` — Tile-based parallel renderer, ray tracer with observer + source Doppler
- `pkg/retarded/` — Retarded-time solver for moving objects (Newton's method)
- `pkg/scenefile/` — YAML scene file loader with velocity validation
- `pkg/output/` — PNG writer, FFmpeg video assembly

## Architecture Principles

- **Physically correct**: Lorentz transformations apply to rays natively; relativistic effects are not approximated or composited
- **Spectral rendering**: objects carry full-spectrum reflectivity/emission data so Doppler wavelength shifts produce correct colors
- **Special relativity only**: flat spacetime, straight rays — no general relativity
- **Retarded time**: moving objects are rendered at their position when the light left them, solved iteratively
- **Source + observer Doppler**: both the camera's velocity and the object's velocity contribute Doppler shift
- **Reference implementation**: correctness over performance; this Go version validates the physics before any future GPU port
