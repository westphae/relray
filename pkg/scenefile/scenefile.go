package scenefile

import (
	"fmt"
	"math"
	"os"

	"gopkg.in/yaml.v3"

	"sif/gogs/eric/relray/pkg/camera"
	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/material"
	"sif/gogs/eric/relray/pkg/retarded"
	"sif/gogs/eric/relray/pkg/scene"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Load reads a YAML scene file and returns the scene and optional camera.
// If the file doesn't define a camera, cam will be nil.
func Load(path string) (*scene.Scene, *camera.Camera, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading scene file: %w", err)
	}

	var sf SceneFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, nil, fmt.Errorf("parsing scene file: %w", err)
	}

	return convert(&sf)
}

func convert(sf *SceneFile) (*scene.Scene, *camera.Camera, error) {
	sc := &scene.Scene{Name: sf.Name}

	// Lights
	for i, ls := range sf.Lights {
		spd, err := convertSPD(&ls.Emission)
		if err != nil {
			return nil, nil, fmt.Errorf("lights[%d].emission: %w", i, err)
		}
		sc.Lights = append(sc.Lights, scene.Light{
			Position: v3(ls.Position),
			Emission: spd,
		})
	}

	// Static objects
	for i, os := range sf.Objects {
		shape, err := convertShape(&os.Shape)
		if err != nil {
			return nil, nil, fmt.Errorf("objects[%d].shape: %w", i, err)
		}
		mat, err := convertMaterial(&os.Material)
		if err != nil {
			return nil, nil, fmt.Errorf("objects[%d].material: %w", i, err)
		}
		sc.Objects = append(sc.Objects, scene.Object{Shape: shape, Material: mat})
	}

	// Moving objects
	for i, ms := range sf.MovingObjects {
		shape, err := convertShape(&ms.Shape)
		if err != nil {
			return nil, nil, fmt.Errorf("moving_objects[%d].shape: %w", i, err)
		}
		mat, err := convertMaterial(&ms.Material)
		if err != nil {
			return nil, nil, fmt.Errorf("moving_objects[%d].material: %w", i, err)
		}
		traj, err := convertTrajectory(&ms.Trajectory, i)
		if err != nil {
			return nil, nil, fmt.Errorf("moving_objects[%d].trajectory: %w", i, err)
		}
		sc.MovingObjects = append(sc.MovingObjects, scene.MovingObject{
			Shape: shape, Material: mat, Trajectory: traj,
		})
	}

	// Sky
	if sf.Sky != nil {
		skyFn, err := convertSky(sf.Sky)
		if err != nil {
			return nil, nil, fmt.Errorf("sky: %w", err)
		}
		sc.Sky = skyFn
	} else {
		sc.Sky = func(dir vec.Vec3) spectrum.SPD { return spectrum.SPD{} }
	}

	// Camera
	var cam *camera.Camera
	if sf.Camera != nil {
		c := sf.Camera
		cam = &camera.Camera{
			Position: v3(c.Position),
			LookAt:   v3(c.LookAt),
			Up:       v3(c.Up),
			VFOV:     c.VFOV,
		}
	}

	return sc, cam, nil
}

// --- SPD ---

func convertSPD(s *SPDSpec) (spectrum.SPD, error) {
	if s == nil {
		return spectrum.SPD{}, fmt.Errorf("missing SPD specification")
	}
	switch {
	case s.RGB != nil:
		return spectrum.FromRGB(s.RGB[0], s.RGB[1], s.RGB[2]), nil
	case s.Blackbody != nil:
		return spectrum.Blackbody(s.Blackbody.Temp, s.Blackbody.Luminance), nil
	case s.Constant != nil:
		return spectrum.Constant(*s.Constant), nil
	case s.D65 != nil:
		return spectrum.D65().Scale(*s.D65), nil
	case s.Monochromatic != nil:
		return spectrum.Monochromatic(s.Monochromatic.Wavelength, s.Monochromatic.Power), nil
	case s.Reflectance != nil:
		return spectrum.FromReflectanceCurve(*s.Reflectance), nil
	default:
		return spectrum.SPD{}, fmt.Errorf("no SPD type specified (use rgb, blackbody, constant, d65, monochromatic, or reflectance)")
	}
}

// --- Shapes ---

func convertShape(s *ShapeSpec) (geometry.Shape, error) {
	var shape geometry.Shape
	var err error

	switch {
	case s.Sphere != nil:
		shape = &geometry.Sphere{Radius: s.Sphere.Radius}
	case s.Plane != nil:
		shape = &geometry.Plane{}
	case s.Box != nil:
		shape = &geometry.Box{Size: v3(s.Box.Size)}
	case s.Cylinder != nil:
		shape = &geometry.Cylinder{Radius: s.Cylinder.Radius, Height: s.Cylinder.Height}
	case s.Cone != nil:
		shape = &geometry.Cone{Radius: s.Cone.Radius, Height: s.Cone.Height}
	case s.Disk != nil:
		shape = &geometry.Disk{Radius: s.Disk.Radius}
	case s.Triangle != nil:
		shape = &geometry.Triangle{V0: v3(s.Triangle.V0), V1: v3(s.Triangle.V1), V2: v3(s.Triangle.V2)}
	case s.Pyramid != nil:
		shape = &geometry.Pyramid{BaseRadius: s.Pyramid.BaseRadius, Height: s.Pyramid.Height, Sides: s.Pyramid.Sides}
	default:
		err = fmt.Errorf("no shape type specified (use sphere, plane, box, cylinder, cone, disk, triangle, or pyramid)")
	}
	if err != nil {
		return nil, err
	}

	// Apply optional transform
	if s.Position != nil || s.Rotation != nil {
		pos := vec.Vec3{}
		rot := vec.Identity()
		if s.Position != nil {
			pos = v3(*s.Position)
		}
		if s.Rotation != nil {
			rot = vec.RotationFromEulerDeg(s.Rotation[0], s.Rotation[1], s.Rotation[2])
		}
		shape = geometry.NewTransformed(shape, pos, rot)
	}

	return shape, nil
}

// --- Materials ---

func convertMaterial(m *MaterialSpec) (material.Material, error) {
	switch {
	case m.Diffuse != nil:
		spd, err := convertSPD(&m.Diffuse.SPDSpec)
		if err != nil {
			return nil, fmt.Errorf("diffuse: %w", err)
		}
		return &material.Diffuse{Reflectance: spd}, nil
	case m.Mirror != nil:
		spd, err := convertSPD(&m.Mirror.SPDSpec)
		if err != nil {
			return nil, fmt.Errorf("mirror: %w", err)
		}
		return &material.Mirror{Reflectance: spd}, nil
	case m.Glass != nil:
		tint, err := convertSPD(&m.Glass.Tint)
		if err != nil {
			return nil, fmt.Errorf("glass.tint: %w", err)
		}
		return &material.Glass{IOR: m.Glass.IOR, Tint: tint}, nil
	case m.Checker != nil:
		even, err := convertSPD(&m.Checker.Even)
		if err != nil {
			return nil, fmt.Errorf("checker.even: %w", err)
		}
		odd, err := convertSPD(&m.Checker.Odd)
		if err != nil {
			return nil, fmt.Errorf("checker.odd: %w", err)
		}
		return &material.Checker{Even: even, Odd: odd, Scale: m.Checker.Scale}, nil
	case m.CheckerSphere != nil:
		even, err := convertSPD(&m.CheckerSphere.Even)
		if err != nil {
			return nil, fmt.Errorf("checker_sphere.even: %w", err)
		}
		odd, err := convertSPD(&m.CheckerSphere.Odd)
		if err != nil {
			return nil, fmt.Errorf("checker_sphere.odd: %w", err)
		}
		return &material.CheckerSphere{Even: even, Odd: odd, NumSquares: m.CheckerSphere.NumSquares}, nil
	default:
		return nil, fmt.Errorf("no material type specified (use diffuse, mirror, glass, checker, or checker_sphere)")
	}
}

// --- Trajectories ---

func convertTrajectory(t *TrajectorySpec, _ int) (retarded.Trajectory, error) {
	switch {
	case t.Static != nil:
		pos := v3(t.Static.Position)
		return func(float64) vec.Vec3 { return pos }, nil

	case t.Linear != nil:
		start := v3(t.Linear.Start)
		vel := v3(t.Linear.Velocity)
		speed := vel.Length()
		if speed >= retarded.C {
			return nil, fmt.Errorf("linear: speed %.3f exceeds c (%.1f)", speed, retarded.C)
		}
		return func(tm float64) vec.Vec3 { return start.Add(vel.Scale(tm)) }, nil

	case t.Orbit != nil:
		o := t.Orbit
		maxSpeed := 2 * math.Pi * o.Radius / o.Period
		if maxSpeed >= retarded.C {
			return nil, fmt.Errorf("orbit: max speed %.3f exceeds c (%.1f)", maxSpeed, retarded.C)
		}
		center := v3(o.Center)
		return makeOrbitTrajectory(center, o.Radius, o.Period, o.Axis), nil

	default:
		return nil, fmt.Errorf("no trajectory type specified (use static, linear, or orbit)")
	}
}

func makeOrbitTrajectory(center vec.Vec3, radius, period float64, axis string) retarded.Trajectory {
	return func(t float64) vec.Vec3 {
		angle := 2 * math.Pi * t / period
		cos, sin := math.Cos(angle), math.Sin(angle)
		switch axis {
		case "x":
			return vec.Vec3{X: center.X, Y: center.Y + radius*cos, Z: center.Z + radius*sin}
		case "z":
			return vec.Vec3{X: center.X + radius*cos, Y: center.Y + radius*sin, Z: center.Z}
		default: // "y" or unspecified
			return vec.Vec3{X: center.X + radius*cos, Y: center.Y, Z: center.Z + radius*sin}
		}
	}
}

// --- Sky ---

func convertSky(s *SkySpec) (func(vec.Vec3) spectrum.SPD, error) {
	switch s.Type {
	case "none", "":
		return func(vec.Vec3) spectrum.SPD { return spectrum.SPD{} }, nil

	case "uniform":
		if s.Emission == nil {
			return nil, fmt.Errorf("uniform sky requires 'emission'")
		}
		spd, err := convertSPD(s.Emission)
		if err != nil {
			return nil, err
		}
		return func(vec.Vec3) spectrum.SPD { return spd }, nil

	case "gradient":
		if s.Top == nil || s.Bottom == nil {
			return nil, fmt.Errorf("gradient sky requires 'top' and 'bottom'")
		}
		top, err := convertSPD(s.Top)
		if err != nil {
			return nil, fmt.Errorf("top: %w", err)
		}
		bot, err := convertSPD(s.Bottom)
		if err != nil {
			return nil, fmt.Errorf("bottom: %w", err)
		}
		return func(dir vec.Vec3) spectrum.SPD {
			t := 0.5 * (dir.Y + 1.0)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			return bot.Scale(1 - t).Add(top.Scale(t))
		}, nil

	default:
		return nil, fmt.Errorf("unknown sky type %q (use none, uniform, or gradient)", s.Type)
	}
}

// --- Helpers ---

func v3(a [3]float64) vec.Vec3 {
	return vec.Vec3{X: a[0], Y: a[1], Z: a[2]}
}
