package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	flag "github.com/spf13/pflag"

	"sif/gogs/eric/relray/pkg/camera"
	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/material"
	"sif/gogs/eric/relray/pkg/output"
	"sif/gogs/eric/relray/pkg/render"
	"sif/gogs/eric/relray/pkg/scene"
	"sif/gogs/eric/relray/pkg/scenefile"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// commonFlags holds the parsed values from flags shared by all subcommands.
type commonFlags struct {
	width, height, samples, depth int
	sceneName, file, out          string
}

func addCommonFlags(fs *flag.FlagSet) *commonFlags {
	cf := &commonFlags{}
	fs.IntVar(&cf.width, "width", 800, "image width")
	fs.IntVar(&cf.height, "height", 600, "image height")
	fs.IntVar(&cf.samples, "samples", 32, "samples per pixel")
	fs.IntVar(&cf.depth, "depth", 8, "max ray bounces")
	fs.StringVar(&cf.sceneName, "scene", "spheres", "built-in scene: spheres, room")
	fs.StringVar(&cf.file, "file", "", "load scene from YAML file (overrides --scene)")
	fs.StringVar(&cf.out, "out", "", "output filename")
	return cf
}

func (cf *commonFlags) config() render.Config {
	return render.Config{
		Width:        cf.width,
		Height:       cf.height,
		MaxDepth:     cf.depth,
		SamplesPerPx: cf.samples,
	}
}

// loadScene loads from --file if provided, otherwise uses the built-in --scene.
func (cf *commonFlags) loadScene() (*scene.Scene, *camera.Camera) {
	return cf.loadSceneWithVars(nil)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	subcmd := os.Args[1]

	// Top-level help
	if subcmd == "-h" || subcmd == "--help" || subcmd == "help" {
		printUsage()
		os.Exit(0)
	}

	// If the first arg looks like a flag, treat it as "render"
	if len(subcmd) > 0 && subcmd[0] == '-' {
		subcmd = "render"
		os.Args = append([]string{os.Args[0], "render"}, os.Args[1:]...)
	}

	switch subcmd {
	case "render":
		fs := flag.NewFlagSet("render", flag.ExitOnError)
		cf := addCommonFlags(fs)
		varFlags := fs.StringArray("var", nil, "set variable: name=value (repeatable)")
		fs.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: relray render [flags]\n\nRender a single static image.\n\nFlags:\n")
			fs.PrintDefaults()
		}
		fs.Parse(os.Args[2:])

		vars := parseVarFlags(*varFlags)
		sc, cam := cf.loadSceneWithVars(vars)
		if cf.out == "" {
			cf.out = "output.png"
		}
		if cam == nil {
			cam = cameraPreset(sc.Name, cf.width, cf.height)
		}
		cam.Aspect = float64(cf.width) / float64(cf.height)
		cam.Init()
		runSingle(cf.config(), sc, cam, cf.out)

	case "sweep":
		fs := flag.NewFlagSet("sweep", flag.ExitOnError)
		cf := addCommonFlags(fs)
		rangeFlags := fs.StringArray("range", nil, "sweep variable: name:start:end (repeatable)")
		steps := fs.Int("steps", 200, "number of frames")
		fps := fs.Int("fps", 30, "video framerate")
		fs.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: relray sweep [flags]\n\nRender a video sweeping variables across a range.\n\nFlags:\n")
			fs.PrintDefaults()
		}
		fs.Parse(os.Args[2:])

		if cf.file == "" {
			log.Fatal("sweep requires --file with a YAML scene containing $variables")
		}
		ranges := parseRangeFlags(*rangeFlags)
		if cf.out == "" {
			cf.out = "sweep.mp4"
		}
		runSweep(cf.config(), cf.file, cf.width, cf.height, ranges, *steps, *fps, cf.out)

	case "walk":
		fs := flag.NewFlagSet("walk", flag.ExitOnError)
		cf := addCommonFlags(fs)
		duration := fs.Float64("duration", 10.0, "walk duration in seconds")
		speed := fs.Float64("speed", 0.5, "observer speed as fraction of c")
		fps := fs.Int("fps", 30, "video framerate")
		fs.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: relray walk [flags]\n\nRender a first-person walk-through video.\n\nFlags:\n")
			fs.PrintDefaults()
		}
		fs.Parse(os.Args[2:])

		sc, cam := cf.loadScene()
		if cf.out == "" {
			cf.out = "walk.mp4"
		}
		runWalk(cf.config(), sc, cam, cf.width, cf.height, *duration, *speed, *fps, cf.out)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", subcmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: relray <command> [flags]

Relativistic ray tracer — renders scenes with physically correct
aberration, Doppler shift, and searchlight effects.

Commands:
  render    Render a single static image (default)
  sweep     Render a video sweeping variables across a range
  walk      Render a first-person walk-through video

Common flags (all commands):
  --width int       image width (default 800)
  --height int      image height (default 600)
  --samples int     samples per pixel (default 32)
  --depth int       max ray bounces (default 8)
  --scene string    built-in scene: spheres, room (default "spheres")
  --file string     load scene from YAML file (overrides --scene)
  --out string      output filename

Run 'relray <command> --help' for command-specific flags.
`)
}

// loadSceneWithVars loads a scene from --file with variable substitution,
// or falls back to a built-in scene.
func (cf *commonFlags) loadSceneWithVars(vars map[string]float64) (*scene.Scene, *camera.Camera) {
	if cf.file != "" {
		sc, cam, err := scenefile.LoadWithVars(cf.file, vars)
		if err != nil {
			log.Fatalf("loading scene file: %v", err)
		}
		return sc, cam
	}
	switch cf.sceneName {
	case "room":
		return buildRoomScene(), nil
	default:
		return buildSpheresScene(), nil
	}
}

// parseVarFlags parses --var flags into a map.
func parseVarFlags(flags []string) map[string]float64 {
	if len(flags) == 0 {
		return nil
	}
	vars := make(map[string]float64, len(flags))
	for _, f := range flags {
		name, val, err := scenefile.ParseVar(f)
		if err != nil {
			log.Fatal(err)
		}
		vars[name] = val
	}
	return vars
}

// parseRangeFlags parses --range flags into VarRange slices.
func parseRangeFlags(flags []string) []scenefile.VarRange {
	ranges := make([]scenefile.VarRange, 0, len(flags))
	for _, f := range flags {
		r, err := scenefile.ParseRange(f)
		if err != nil {
			log.Fatal(err)
		}
		ranges = append(ranges, r)
	}
	return ranges
}

// cameraPreset returns a default camera for the given built-in scene name.
func cameraPreset(sceneName string, width, height int) *camera.Camera {
	aspect := float64(width) / float64(height)
	switch sceneName {
	case "room":
		return &camera.Camera{
			Position: vec.Vec3{X: 0, Y: 1.0, Z: -0.5},
			LookAt:   vec.Vec3{X: 0, Y: 0.8, Z: 3.0},
			Up:       vec.Vec3{Y: 1},
			VFOV:     70,
			Aspect:   aspect,
		}
	default:
		return &camera.Camera{
			Position: vec.Vec3{X: 0, Y: 0.5, Z: -3},
			LookAt:   vec.Vec3{X: 0, Y: 0.3, Z: 0},
			Up:       vec.Vec3{Y: 1},
			VFOV:     60,
			Aspect:   aspect,
		}
	}
}

func runSingle(cfg render.Config, sc *scene.Scene, cam *camera.Camera, outFile string) {
	v := cam.Velocity
	fmt.Printf("Rendering %dx%d, velocity=[%.3f,%.3f,%.3f], %d spp, %d bounces\n",
		cfg.Width, cfg.Height, v.X, v.Y, v.Z, cfg.SamplesPerPx, cfg.MaxDepth)

	start := time.Now()
	img := render.RenderFrame(cfg, sc, cam)
	fmt.Printf("Rendered in %v\n", time.Since(start))

	if err := output.SavePNG(outFile, img); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved to %s\n", outFile)
}

func runSweep(cfg render.Config, file string, width, height int, ranges []scenefile.VarRange, steps, fps int, outFile string) {
	if outFile == "" {
		outFile = "sweep.mp4"
	}

	for _, r := range ranges {
		fmt.Printf("  %s: %.4f → %.4f\n", r.Name, r.Start, r.End)
	}
	fmt.Printf("Sweep: %d steps, %dx%d, %d spp, %d bounces, %d fps\n",
		steps, cfg.Width, cfg.Height, cfg.SamplesPerPx, cfg.MaxDepth, fps)

	frameDir, err := os.MkdirTemp("", "relray-sweep-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(frameDir)

	aspect := float64(width) / float64(height)

	totalStart := time.Now()
	for i := range steps {
		t := 0.0
		if steps > 1 {
			t = float64(i) / float64(steps-1)
		}
		vars := scenefile.InterpolateVars(ranges, t)

		sc, cam, err := scenefile.LoadWithVars(file, vars)
		if err != nil {
			log.Fatalf("frame %d: %v", i, err)
		}
		if cam == nil {
			log.Fatal("sweep requires a camera defined in the YAML scene file")
		}
		cam.Aspect = aspect
		cam.Init()

		start := time.Now()
		img := render.RenderFrame(cfg, sc, cam)
		elapsed := time.Since(start)

		framePath := filepath.Join(frameDir, fmt.Sprintf("frame_%04d.png", i))
		if err := output.SavePNG(framePath, img); err != nil {
			log.Fatal(err)
		}

		varStr := ""
		for _, r := range ranges {
			varStr += fmt.Sprintf("  %s=%+.4f", r.Name, vars[r.Name])
		}
		fmt.Printf("Frame %d/%d%s  %v\n", i+1, steps, varStr, elapsed)
	}

	fmt.Printf("All frames rendered in %v\n", time.Since(totalStart))
	fmt.Printf("Assembling video...\n")

	pattern := filepath.Join(frameDir, "frame_%04d.png")
	if err := output.AssembleVideo(pattern, fps, outFile); err != nil {
		log.Fatalf("ffmpeg failed: %v", err)
	}
	fmt.Printf("Saved to %s\n", outFile)
}

func runWalk(cfg render.Config, sc *scene.Scene, fileCam *camera.Camera, width, height int, duration, speed float64, fps int, outFile string) {
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

	startZ := -2.0
	eyeY := 1.0
	if fileCam != nil {
		startZ = fileCam.Position.Z
		eyeY = fileCam.Position.Y
	}

	totalStart := time.Now()
	for i := range numFrames {
		t := float64(i) * dt
		z := startZ + speed*t

		sc.Time = t

		cam := &camera.Camera{
			Position: vec.Vec3{X: 0, Y: eyeY, Z: z},
			LookAt:   vec.Vec3{X: 0, Y: eyeY - 0.1, Z: z + 2},
			Up:       vec.Vec3{Y: 1},
			VFOV:     70,
			Aspect:   float64(width) / float64(height),
			Velocity: vec.Vec3{Z: speed},
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

// at returns a shape positioned at the given coordinates (no rotation).
func at(shape geometry.Shape, x, y, z float64) geometry.Shape {
	return geometry.NewTransformed(shape, vec.Vec3{X: x, Y: y, Z: z}, vec.Identity())
}

// atRot returns a shape positioned and rotated (Euler angles in degrees).
func atRot(shape geometry.Shape, x, y, z, yaw, pitch, roll float64) geometry.Shape {
	return geometry.NewTransformed(shape, vec.Vec3{X: x, Y: y, Z: z}, vec.RotationFromEulerDeg(yaw, pitch, roll))
}

// box creates a box shape with given width, height, depth positioned at the given center.
func box(w, h, d, cx, cy, cz float64) geometry.Shape {
	return at(&geometry.Box{Size: vec.Vec3{X: w, Y: h, Z: d}}, cx, cy, cz)
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
			{Shape: at(&geometry.Plane{}, 0, -0.5, 0), Material: floor},
			{Shape: at(&geometry.Sphere{Radius: 0.5}, -1.8, 0, 1.5), Material: red},
			{Shape: at(&geometry.Sphere{Radius: 0.5}, -0.6, 0, 2), Material: green},
			{Shape: at(&geometry.Sphere{Radius: 0.5}, 0.6, 0, 2), Material: mirror},
			{Shape: at(&geometry.Sphere{Radius: 0.5}, 1.8, 0, 1.5), Material: glass},
			{Shape: at(&geometry.Sphere{Radius: 0.2}, 0, -0.3, 1), Material: blue},
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
			// Floor (Y=0, normal +Y — default plane orientation)
			{Shape: at(&geometry.Plane{}, 0, 0, 0), Material: floorWood},
			// Ceiling (Y=2.5, normal -Y — flip plane 180°)
			{Shape: atRot(&geometry.Plane{}, 0, 2.5, 0, 0, 0, 180), Material: ceiling},
			// Back wall (Z=6, normal -Z — rotate plane -90° pitch)
			{Shape: atRot(&geometry.Plane{}, 0, 0, 6, 0, -90, 0), Material: wallAccent},
			// Left wall (X=-3, normal +X — rotate plane 90° roll)
			{Shape: atRot(&geometry.Plane{}, -3, 0, 0, 0, 0, 90), Material: wallWhite},
			// Right wall (X=3, normal -X — rotate plane -90° roll)
			{Shape: atRot(&geometry.Plane{}, 3, 0, 0, 0, 0, -90), Material: wallWhite},
			// Front wall (Z=-2, normal +Z — rotate plane 90° pitch)
			{Shape: atRot(&geometry.Plane{}, 0, 0, -2, 0, 90, 0), Material: wallWhite},

			// Coffee table: 1.0 x 0.4 x 1.0, centered at (0, 0.2, 3.0)
			{Shape: box(1.0, 0.4, 1.0, 0, 0.2, 3.0), Material: tableMat},
			// Couch base: 1.3 x 0.45 x 3.0, centered at (-2.15, 0.225, 3.0)
			{Shape: box(1.3, 0.45, 3.0, -2.15, 0.225, 3.0), Material: furniture},
			// Couch back: 0.3 x 0.45 x 3.0, centered at (-2.65, 0.675, 3.0)
			{Shape: box(0.3, 0.45, 3.0, -2.65, 0.675, 3.0), Material: furniture},
			// Couch cushion: 0.9 x 0.1 x 2.6, centered at (-2.05, 0.5, 3.0)
			{Shape: box(0.9, 0.1, 2.6, -2.05, 0.5, 3.0), Material: cushion},
			// Bookshelf: 1.0 x 1.8 x 1.8, centered at (2.3, 0.9, 4.9)
			{Shape: box(1.0, 1.8, 1.8, 2.3, 0.9, 4.9), Material: furniture},

			// Glass globe on coffee table
			{Shape: at(&geometry.Sphere{Radius: 0.12}, 0.1, 0.55, 3.0), Material: glassMat},
			// Small mirror sphere on coffee table
			{Shape: at(&geometry.Sphere{Radius: 0.08}, -0.2, 0.52, 2.8), Material: mirrorMat},
			// Red decorative ball
			{Shape: at(&geometry.Sphere{Radius: 0.08}, 0.3, 0.5, 3.2), Material: ballMat},
		},

		// Orbiting globe above the coffee table, suspended from ceiling.
		// Completes one orbit per walk duration (10s default).
		// Orbit radius 0.4m gives speed ≈ 2π·0.4/10 ≈ 0.25c — enough for visible effects.
		MovingObjects: []scene.MovingObject{
			{
				Shape:    &geometry.Sphere{Radius: 0.12},
				Material: &material.CheckerSphere{
					Even:       spectrum.FromRGB(0.9, 0.85, 0.15),
					Odd:        spectrum.FromRGB(0.1, 0.1, 0.6),
					NumSquares: 8,
				},
				Trajectory: func(t float64) vec.Vec3 {
					const (
						orbitRadius = 0.4
						orbitPeriod = 10.0
						centerX     = 0.0
						centerZ     = 3.0
						height      = 1.2 // hanging below ceiling
					)
					angle := 2 * math.Pi * t / orbitPeriod
					return vec.Vec3{
						X: centerX + orbitRadius*math.Cos(angle),
						Y: height,
						Z: centerZ + orbitRadius*math.Sin(angle),
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
