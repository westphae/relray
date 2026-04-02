package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"sif/gogs/eric/relray/pkg/camera"
	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/material"
	"sif/gogs/eric/relray/pkg/output"
	"sif/gogs/eric/relray/pkg/render"
	"sif/gogs/eric/relray/pkg/scene"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

func main() {
	width := flag.Int("width", 800, "image width")
	height := flag.Int("height", 600, "image height")
	beta := flag.Float64("beta", 0.0, "observer speed as fraction of c (along +Z)")
	samples := flag.Int("samples", 4, "samples per pixel")
	depth := flag.Int("depth", 4, "max ray bounces")
	outFile := flag.String("out", "output.png", "output filename")
	flag.Parse()

	sc := buildScene()
	cam := &camera.Camera{
		Position: vec.Vec3{X: 0, Y: 0.5, Z: -3},
		LookAt:   vec.Vec3{X: 0, Y: 0.3, Z: 0},
		Up:       vec.Vec3{Y: 1},
		VFOV:     60,
		Aspect:   float64(*width) / float64(*height),
		Beta:     vec.Vec3{Z: *beta},
	}

	cfg := render.Config{
		Width:        *width,
		Height:       *height,
		MaxDepth:     *depth,
		SamplesPerPx: *samples,
	}

	fmt.Printf("Rendering %dx%d, beta=%.2f, %d spp, %d bounces\n",
		cfg.Width, cfg.Height, *beta, cfg.SamplesPerPx, cfg.MaxDepth)

	start := time.Now()
	img := render.RenderFrame(cfg, sc, cam)
	elapsed := time.Since(start)
	fmt.Printf("Rendered in %v\n", elapsed)

	if err := output.SavePNG(*outFile, img); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved to %s\n", *outFile)
}

func buildScene() *scene.Scene {
	// D65 daylight for the light source
	d65 := spectrum.D65()

	// Materials: three colored spheres + gray floor
	red := &material.Diffuse{Reflectance: spectrum.FromRGB(0.8, 0.1, 0.1)}
	green := &material.Diffuse{Reflectance: spectrum.FromRGB(0.1, 0.8, 0.1)}
	blue := &material.Diffuse{Reflectance: spectrum.FromRGB(0.1, 0.1, 0.8)}
	floor := &material.Diffuse{Reflectance: spectrum.FromRGB(0.5, 0.5, 0.5)}

	sc := &scene.Scene{
		Objects: []scene.Object{
			// Floor plane at y = -0.5
			{Shape: &geometry.Plane{Point: vec.Vec3{Y: -0.5}, Normal: vec.Vec3{Y: 1}}, Material: floor},
			// Red sphere (left)
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: -1.2, Y: 0, Z: 1}, Radius: 0.5}, Material: red},
			// Green sphere (center)
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: 0, Y: 0, Z: 2}, Radius: 0.5}, Material: green},
			// Blue sphere (right)
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: 1.2, Y: 0, Z: 1}, Radius: 0.5}, Material: blue},
		},
		Lights: []scene.Light{
			// Overhead key light
			{Position: vec.Vec3{X: 2, Y: 5, Z: 0}, Emission: d65.Scale(3000)},
			// Fill light from the other side
			{Position: vec.Vec3{X: -3, Y: 3, Z: -2}, Emission: d65.Scale(1500)},
		},
		Sky: func(dir vec.Vec3) spectrum.SPD {
			// Gradient sky: blue overhead fading to darker at horizon
			t := 0.5 * (dir.Y + 1.0)
			if t < 0 {
				t = 0
			}
			return spectrum.FromRGB(0.4*t, 0.5*t, 0.9*t)
		},
	}
	return sc
}
