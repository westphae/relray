# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**relray** is a relativistic ray tracer that renders video showing what a human would see moving through a familiar environment where the speed of light is reduced to a human scale (~1m/s). All relativistic visual effects (length contraction, time dilation, aberration, Doppler shift, searchlight effect) emerge naturally from the ray tracing math — nothing is added as post-processing.

See GOAL.md for the full project vision.

## Language & Build

- Language: Go
- Build: `go build ./...`
- Test: `go test ./...`
- Single test: `go test ./pkg/... -run TestName`
- Target hardware: 64-core CPU, 256GB RAM, NVIDIA RTX 4070 Ti SUPER (16GB VRAM)
- GPU is not used initially; the Go implementation is CPU-only, parallelized across all cores
- A future C++/CUDA/OptiX rewrite of the render kernel is planned once physics are validated

## Architecture Principles

- **Physically correct**: Lorentz transformations apply to rays natively; relativistic effects are not approximated or composited
- **Spectral rendering**: objects carry full-spectrum reflectivity/emission data so Doppler wavelength shifts produce correct colors
- **Special relativity only**: flat spacetime, straight rays — no general relativity
- **Retarded time**: moving objects are rendered at their position when the light left them, solved iteratively
- **Modular**: simple scenes first, complexity added incrementally; the codebase should support this growth
- **Reference implementation**: correctness over performance; this Go version validates the physics before any future GPU port
