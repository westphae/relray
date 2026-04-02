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
	sceneFlag := flag.String("scene", "spheres", "scene to render: spheres, room")

	sweep := flag.Bool("sweep", false, "render beta sweep video")
	betaMin := flag.Float64("beta-min", -0.5, "sweep: starting beta")
	betaMax := flag.Float64("beta-max", 0.5, "sweep: ending beta")
	betaStep := flag.Float64("beta-step", 0.001, "sweep: beta increment per frame")
	fps := flag.Int("fps", 30, "sweep/walk: video framerate")

	walk := flag.Bool("walk", false, "render walk-through video of the room scene")
	walkDuration := flag.Float64("walk-duration", 10.0, "walk: duration in seconds")
	walkSpeed := flag.Float64("walk-speed", 0.5, "walk: observer speed in scene units/s (fraction of c)")
	flag.Parse()

	var sc *scene.Scene
	switch *sceneFlag {
	case "room":
		sc = buildRoomScene()
	default:
		sc = buildSpheresScene()
	}

	cfg := render.Config{
		Width:        *width,
		Height:       *height,
		MaxDepth:     *depth,
		SamplesPerPx: *samples,
	}

	if *walk {
		runWalk(cfg, sc, *width, *height, *walkDuration, *walkSpeed, *fps, *outFile)
	} else if *sweep {
		runSweep(cfg, sc, *width, *height, *betaMin, *betaMax, *betaStep, *fps, *outFile)
	} else {
		runSingle(cfg, sc, *width, *height, *beta, *outFile)
	}
}

// CameraPreset returns a default camera for the given scene name.
func cameraPreset(sceneName string, width, height int, beta float64) *camera.Camera {
	aspect := float64(width) / float64(height)
	switch sceneName {
	case "room":
		return &camera.Camera{
			Position: vec.Vec3{X: 0, Y: 1.0, Z: -0.5},
			LookAt:   vec.Vec3{X: 0, Y: 0.8, Z: 3.0},
			Up:       vec.Vec3{Y: 1},
			VFOV:     70,
			Aspect:   aspect,
			Beta:     vec.Vec3{Z: beta},
		}
	default:
		return &camera.Camera{
			Position: vec.Vec3{X: 0, Y: 0.5, Z: -3},
			LookAt:   vec.Vec3{X: 0, Y: 0.3, Z: 0},
			Up:       vec.Vec3{Y: 1},
			VFOV:     60,
			Aspect:   aspect,
			Beta:     vec.Vec3{Z: beta},
		}
	}
}

func runSingle(cfg render.Config, sc *scene.Scene, width, height int, beta float64, outFile string) {
	if outFile == "" {
		outFile = "output.png"
	}
	cam := cameraPreset(sc.Name, width, height, beta)

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

	numFrames := int(math.Round((betaMax-betaMin)/betaStep)) + 1
	fmt.Printf("Beta sweep: %.3f to %.3f, step %.4f (%d frames)\n", betaMin, betaMax, betaStep, numFrames)
	fmt.Printf("Rendering %dx%d, %d spp, %d bounces, %d fps\n",
		cfg.Width, cfg.Height, cfg.SamplesPerPx, cfg.MaxDepth, fps)

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

		cam := cameraPreset(sc.Name, width, height, b)

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

func runWalk(cfg render.Config, sc *scene.Scene, width, height int, duration, speed float64, fps int, outFile string) {
	if outFile == "" {
		outFile = "walk.mp4"
	}

	numFrames := int(duration * float64(fps))
	dt := 1.0 / float64(fps)
	fmt.Printf("Walk-through: %.1fs at speed %.2f c, %d frames\n", duration, speed, numFrames)
	fmt.Printf("Rendering %dx%d, %d spp, %d bounces, %d fps\n",
		cfg.Width, cfg.Height, cfg.SamplesPerPx, cfg.MaxDepth, fps)

	frameDir, err := os.MkdirTemp("", "relray-walk-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(frameDir)

	// Camera path: walk along +Z through the room
	// Start position and look-ahead distance
	startZ := -2.0
	eyeY := 1.0 // eye height

	totalStart := time.Now()
	for i := range numFrames {
		t := float64(i) * dt
		z := startZ + speed*t

		// Update scene time for moving objects
		sc.Time = t

		cam := &camera.Camera{
			Position: vec.Vec3{X: 0, Y: eyeY, Z: z},
			LookAt:   vec.Vec3{X: 0, Y: eyeY - 0.1, Z: z + 2},
			Up:       vec.Vec3{Y: 1},
			VFOV:     70,
			Aspect:   float64(width) / float64(height),
			Beta:     vec.Vec3{Z: speed},
		}

		start := time.Now()
		img := render.RenderFrame(cfg, sc, cam)
		elapsed := time.Since(start)

		framePath := filepath.Join(frameDir, fmt.Sprintf("frame_%05d.png", i))
		if err := output.SavePNG(framePath, img); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Frame %d/%d  t=%.2fs  z=%.2f  %v\n", i+1, numFrames, t, z, elapsed)
	}

	fmt.Printf("All frames rendered in %v\n", time.Since(totalStart))
	fmt.Printf("Assembling video...\n")

	pattern := filepath.Join(frameDir, "frame_%05d.png")
	if err := output.AssembleVideo(pattern, fps, outFile); err != nil {
		log.Fatalf("ffmpeg failed: %v", err)
	}
	fmt.Printf("Saved to %s\n", outFile)
}

// buildSpheresScene creates the original test scene with three colored spheres.
func buildSpheresScene() *scene.Scene {
	sunlight := spectrum.Blackbody(5778, 1.0)
	fillLight := spectrum.Blackbody(7500, 1.0)
	skyBase := spectrum.Blackbody(12000, 1.0)

	red := &material.Diffuse{Reflectance: spectrum.FromRGB(0.8, 0.1, 0.1)}
	green := &material.Diffuse{Reflectance: spectrum.FromRGB(0.1, 0.8, 0.1)}
	blue := &material.Diffuse{Reflectance: spectrum.FromRGB(0.1, 0.1, 0.8)}
	mirror := &material.Mirror{Reflectance: spectrum.Constant(0.95)}
	glass := &material.Glass{IOR: 1.5, Tint: spectrum.Constant(1.0)}
	floor := &material.Checker{
		Even:  spectrum.FromRGB(0.7, 0.7, 0.7),
		Odd:   spectrum.FromRGB(0.15, 0.15, 0.15),
		Scale: 0.5,
	}

	return &scene.Scene{
		Name: "spheres",
		Objects: []scene.Object{
			{Shape: &geometry.Plane{Point: vec.Vec3{Y: -0.5}, Normal: vec.Vec3{Y: 1}}, Material: floor},
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: -1.8, Y: 0, Z: 1.5}, Radius: 0.5}, Material: red},
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: -0.6, Y: 0, Z: 2}, Radius: 0.5}, Material: green},
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: 0.6, Y: 0, Z: 2}, Radius: 0.5}, Material: mirror},
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: 1.8, Y: 0, Z: 1.5}, Radius: 0.5}, Material: glass},
			{Shape: &geometry.Sphere{Center: vec.Vec3{X: 0, Y: -0.3, Z: 1}, Radius: 0.2}, Material: blue},
		},
		Lights: []scene.Light{
			{Position: vec.Vec3{X: 2, Y: 5, Z: 0}, Emission: sunlight.Scale(15)},
			{Position: vec.Vec3{X: -3, Y: 3, Z: -2}, Emission: fillLight.Scale(8)},
		},
		Sky: func(dir vec.Vec3) spectrum.SPD {
			t := 0.5 * (dir.Y + 1.0)
			if t < 0 {
				t = 0
			}
			return skyBase.Scale(0.15 * t)
		},
	}
}

// buildRoomScene creates a living room scene with walls, floor, furniture,
// a window with sunlight, and a moving sphere.
func buildRoomScene() *scene.Scene {
	sunlight := spectrum.Blackbody(5778, 1.0)
	warmLight := spectrum.Blackbody(3500, 1.0) // warm indoor lamp

	// Room dimensions: 6m wide (X: -3 to 3), 2.5m tall (Y: 0 to 2.5), 8m deep (Z: -2 to 6)
	wallWhite := &material.Diffuse{Reflectance: spectrum.FromRGB(0.85, 0.82, 0.78)}
	wallAccent := &material.Diffuse{Reflectance: spectrum.FromRGB(0.6, 0.15, 0.1)}
	glassMat := &material.Glass{IOR: 1.5, Tint: spectrum.Constant(1.0)}
	mirrorMat := &material.Mirror{Reflectance: spectrum.Constant(0.92)}
	floorWood := &material.Checker{
		Even:  spectrum.FromRGB(0.55, 0.35, 0.18), // dark wood
		Odd:   spectrum.FromRGB(0.65, 0.45, 0.25), // lighter wood
		Scale: 0.4,
	}
	ceiling := &material.Diffuse{Reflectance: spectrum.FromRGB(0.9, 0.9, 0.9)}
	furniture := &material.Diffuse{Reflectance: spectrum.FromRGB(0.3, 0.2, 0.1)}   // dark wood furniture
	cushion := &material.Diffuse{Reflectance: spectrum.FromRGB(0.15, 0.25, 0.5)}   // blue cushion
	tableMat := &material.Diffuse{Reflectance: spectrum.FromRGB(0.4, 0.25, 0.12)}  // table
	ballMat := &material.Diffuse{Reflectance: spectrum.FromRGB(0.9, 0.2, 0.2)}     // red ball

	sc := &scene.Scene{
		Name: "room",
		Objects: []scene.Object{
			// Floor
			{Shape: &geometry.Plane{Point: vec.Vec3{Y: 0}, Normal: vec.Vec3{Y: 1}}, Material: floorWood},
			// Ceiling
			{Shape: &geometry.Plane{Point: vec.Vec3{Y: 2.5}, Normal: vec.Vec3{Y: -1}}, Material: ceiling},
			// Back wall (Z=6)
			{Shape: &geometry.Plane{Point: vec.Vec3{Z: 6}, Normal: vec.Vec3{Z: -1}}, Material: wallAccent},
			// Left wall (X=-3)
			{Shape: &geometry.Plane{Point: vec.Vec3{X: -3}, Normal: vec.Vec3{X: 1}}, Material: wallWhite},
			// Right wall (X=3) — has "window" (we'll fake it with a bright light)
			{Shape: &geometry.Plane{Point: vec.Vec3{X: 3}, Normal: vec.Vec3{X: -1}}, Material: wallWhite},
			// Front wall behind camera (Z=-2)
			{Shape: &geometry.Plane{Point: vec.Vec3{Z: -2}, Normal: vec.Vec3{Z: 1}}, Material: wallWhite},

			// Coffee table (box in center of room)
			{Shape: &geometry.Box{
				Min: vec.Vec3{X: -0.5, Y: 0, Z: 2.5},
				Max: vec.Vec3{X: 0.5, Y: 0.4, Z: 3.5},
			}, Material: tableMat},

			// Couch (left side of room) — simple box
			{Shape: &geometry.Box{
				Min: vec.Vec3{X: -2.8, Y: 0, Z: 1.5},
				Max: vec.Vec3{X: -1.5, Y: 0.45, Z: 4.5},
			}, Material: furniture},
			// Couch back
			{Shape: &geometry.Box{
				Min: vec.Vec3{X: -2.8, Y: 0.45, Z: 1.5},
				Max: vec.Vec3{X: -2.5, Y: 0.9, Z: 4.5},
			}, Material: furniture},
			// Couch cushion
			{Shape: &geometry.Box{
				Min: vec.Vec3{X: -2.5, Y: 0.45, Z: 1.7},
				Max: vec.Vec3{X: -1.6, Y: 0.55, Z: 4.3},
			}, Material: cushion},

			// Bookshelf (right side, against wall)
			{Shape: &geometry.Box{
				Min: vec.Vec3{X: 1.8, Y: 0, Z: 4.0},
				Max: vec.Vec3{X: 2.8, Y: 1.8, Z: 5.8},
			}, Material: furniture},

			// Glass globe on coffee table
			{Shape: &geometry.Sphere{
				Center: vec.Vec3{X: 0.1, Y: 0.55, Z: 3.0},
				Radius: 0.12,
			}, Material: glassMat},
			// Small mirror sphere on coffee table
			{Shape: &geometry.Sphere{
				Center: vec.Vec3{X: -0.2, Y: 0.52, Z: 2.8},
				Radius: 0.08,
			}, Material: mirrorMat},
			// Red decorative ball
			{Shape: &geometry.Sphere{
				Center: vec.Vec3{X: 0.3, Y: 0.5, Z: 3.2},
				Radius: 0.08,
			}, Material: ballMat},
		},

		// Moving object: a ball rolling across the floor
		MovingObjects: []scene.MovingObject{
			{
				Shape:    &geometry.Sphere{Radius: 0.15},
				Material: &material.Diffuse{Reflectance: spectrum.FromRGB(0.2, 0.8, 0.2)},
				Trajectory: func(t float64) vec.Vec3 {
					// Rolling along X from left to right, at Z=2
					return vec.Vec3{
						X: -2.0 + 0.3*t, // 0.3 m/s along X (0.3c!)
						Y: 0.15,
						Z: 2.0,
					}
				},
			},
		},

		Lights: []scene.Light{
			// Window light (sunlight coming from the right)
			{Position: vec.Vec3{X: 2.5, Y: 2.0, Z: 3.0}, Emission: sunlight.Scale(25)},
			// Ceiling lamp
			{Position: vec.Vec3{X: 0, Y: 2.3, Z: 3.0}, Emission: warmLight.Scale(12)},
			// Small lamp on bookshelf
			{Position: vec.Vec3{X: 2.3, Y: 1.9, Z: 4.9}, Emission: warmLight.Scale(5)},
		},

		Sky: func(dir vec.Vec3) spectrum.SPD {
			return spectrum.SPD{} // indoor scene, no sky visible
		},
	}
	return sc
}
