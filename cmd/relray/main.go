package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
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
	outFile := flag.String("out", "", "output filename (default: output.png or sweep.mp4)")

	sweep := flag.Bool("sweep", false, "render beta sweep video")
	betaMin := flag.Float64("beta-min", -0.5, "sweep: starting beta")
	betaMax := flag.Float64("beta-max", 0.5, "sweep: ending beta")
	betaStep := flag.Float64("beta-step", 0.001, "sweep: beta increment per frame")
	fps := flag.Int("fps", 30, "sweep: video framerate")
	flag.Parse()

	sc := buildScene()
	cfg := render.Config{
		Width:        *width,
		Height:       *height,
		MaxDepth:     *depth,
		SamplesPerPx: *samples,
	}

	if *sweep {
		runSweep(cfg, sc, *width, *height, *betaMin, *betaMax, *betaStep, *fps, *outFile)
	} else {
		runSingle(cfg, sc, *width, *height, *beta, *outFile)
	}
}

func runSingle(cfg render.Config, sc *scene.Scene, width, height int, beta float64, outFile string) {
	if outFile == "" {
		outFile = "output.png"
	}
	cam := &camera.Camera{
		Position: vec.Vec3{X: 0, Y: 0.5, Z: -3},
		LookAt:   vec.Vec3{X: 0, Y: 0.3, Z: 0},
		Up:       vec.Vec3{Y: 1},
		VFOV:     60,
		Aspect:   float64(width) / float64(height),
		Beta:     vec.Vec3{Z: beta},
	}

	fmt.Printf("Rendering %dx%d, beta=%.3f, %d spp, %d bounces\n",
		cfg.Width, cfg.Height, beta, cfg.SamplesPerPx, cfg.MaxDepth)

	start := time.Now()
	img := render.RenderFrame(cfg, sc, cam)
	fmt.Printf("Rendered in %v\n", time.Since(start))

	if err := output.SavePNG(outFile, img); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved to %s\n", outFile)
}

func runSweep(cfg render.Config, sc *scene.Scene, width, height int, betaMin, betaMax, betaStep float64, fps int, outFile string) {
	if outFile == "" {
		outFile = "sweep.mp4"
	}

	// Count frames
	numFrames := int(math.Round((betaMax-betaMin)/betaStep)) + 1
	fmt.Printf("Beta sweep: %.3f to %.3f, step %.4f (%d frames)\n", betaMin, betaMax, betaStep, numFrames)
	fmt.Printf("Rendering %dx%d, %d spp, %d bounces, %d fps\n",
		cfg.Width, cfg.Height, cfg.SamplesPerPx, cfg.MaxDepth, fps)

	// Create temp directory for frames
	frameDir, err := os.MkdirTemp("", "relray-sweep-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(frameDir)

	totalStart := time.Now()
	for i := range numFrames {
		b := betaMin + float64(i)*betaStep
		if b > betaMax {
			b = betaMax
		}

		cam := &camera.Camera{
			Position: vec.Vec3{X: 0, Y: 0.5, Z: -3},
			LookAt:   vec.Vec3{X: 0, Y: 0.3, Z: 0},
			Up:       vec.Vec3{Y: 1},
			VFOV:     60,
			Aspect:   float64(width) / float64(height),
			Beta:     vec.Vec3{Z: b},
		}

		start := time.Now()
		img := render.RenderFrame(cfg, sc, cam)
		elapsed := time.Since(start)

		framePath := filepath.Join(frameDir, fmt.Sprintf("frame_%04d.png", i))
		if err := output.SavePNG(framePath, img); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Frame %d/%d  beta=%+.3f  %v\n", i+1, numFrames, b, elapsed)
	}

	fmt.Printf("All frames rendered in %v\n", time.Since(totalStart))
	fmt.Printf("Assembling video...\n")

	pattern := filepath.Join(frameDir, "frame_%04d.png")
	if err := output.AssembleVideo(pattern, fps, outFile); err != nil {
		log.Fatalf("ffmpeg failed: %v", err)
	}
	fmt.Printf("Saved to %s\n", outFile)
}

func buildScene() *scene.Scene {
	// Blackbody light sources
	sunlight := spectrum.Blackbody(5778, 1.0) // solar temperature, warm white
	fillLight := spectrum.Blackbody(7500, 1.0) // cooler fill, slightly blue

	// Sky emission base (blue-tinted blackbody)
	skyBase := spectrum.Blackbody(12000, 1.0)

	// Materials
	red := &material.Diffuse{Reflectance: spectrum.FromRGB(0.8, 0.1, 0.1)}
	green := &material.Diffuse{Reflectance: spectrum.FromRGB(0.1, 0.8, 0.1)}
	blue := &material.Diffuse{Reflectance: spectrum.FromRGB(0.1, 0.1, 0.8)}
	floor := &material.Checker{
		Even:  spectrum.FromRGB(0.7, 0.7, 0.7),
		Odd:   spectrum.FromRGB(0.15, 0.15, 0.15),
		Scale: 0.5,
	}

	sc := &scene.Scene{
		Objects: []scene.Object{
			// Checkerboard floor at y = -0.5
			{Shape: &geometry.Plane{Point: vec.Vec3{Y: -0.5}, Normal: vec.Vec3{Y: 1}}, Material: floor},
			// Red sphere (left)
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: -1.2, Y: 0, Z: 1}, Radius: 0.5}, Material: red},
			// Green sphere (center)
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: 0, Y: 0, Z: 2}, Radius: 0.5}, Material: green},
			// Blue sphere (right)
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: 1.2, Y: 0, Z: 1}, Radius: 0.5}, Material: blue},
		},
		Lights: []scene.Light{
			// Overhead key light (sunlike)
			{Position: vec.Vec3{X: 2, Y: 5, Z: 0}, Emission: sunlight.Scale(15)},
			// Fill light (cooler)
			{Position: vec.Vec3{X: -3, Y: 3, Z: -2}, Emission: fillLight.Scale(8)},
		},
		Sky: func(dir vec.Vec3) spectrum.SPD {
			// Gradient sky using blue-tinted blackbody
			t := 0.5 * (dir.Y + 1.0)
			if t < 0 {
				t = 0
			}
			return skyBase.Scale(0.15 * t)
		},
	}
	return sc
}
