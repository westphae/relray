package render

import (
	"math"

	"sif/gogs/eric/relray/pkg/camera"
	"sif/gogs/eric/relray/pkg/lorentz"
	"sif/gogs/eric/relray/pkg/scene"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Tracer performs ray tracing with relativistic aberration and Doppler shift.
type Tracer struct {
	Scene    *scene.Scene
	Camera   *camera.Camera
	MaxDepth int
}

// Trace traces a single pixel at normalized screen coords (u, v).
func (tr *Tracer) Trace(u, v float64) spectrum.SPD {
	// 1. Generate ray direction in observer's rest frame
	dirObs := tr.Camera.RayDir(u, v)

	// 2. Aberrate to world frame + get Doppler factor
	ab := lorentz.Aberrate(dirObs, tr.Camera.Beta)
	dirWorld := ab.Dir
	doppler := ab.Doppler

	// 3. Trace in world frame
	spd := tr.traceWorld(tr.Camera.Position, dirWorld, tr.MaxDepth)

	// 4. Apply Doppler shift and searchlight effect
	// Shift wavelengths by Doppler factor (blueshift if D > 1)
	// Scale intensity by D^3 (relativistic beaming for surface brightness)
	spd = spd.Shift(1.0 / doppler) // lambda_obs = lambda_emit / D
	spd = spd.Scale(doppler * doppler * doppler)

	return spd
}

// traceWorld traces a ray through the world-frame scene.
func (tr *Tracer) traceWorld(origin, dir vec.Vec3, depth int) spectrum.SPD {
	if depth <= 0 {
		return spectrum.SPD{}
	}

	hit, mat, ok := tr.Scene.Intersect(origin, dir, 0.001, 1e12)
	if !ok {
		// Sky / environment
		if tr.Scene.Sky != nil {
			return tr.Scene.Sky(dir)
		}
		return spectrum.SPD{}
	}

	// Emission from the surface itself
	emitted := mat.Emitted(hit)

	// Direct lighting from point lights
	var direct spectrum.SPD
	for i := range tr.Scene.Lights {
		light := &tr.Scene.Lights[i]
		toLight := light.Position.Sub(hit.Point)
		dist := toLight.Length()
		lightDir := toLight.Scale(1.0 / dist)

		// Shadow test
		if _, _, blocked := tr.Scene.Intersect(hit.Point, lightDir, 0.001, dist-0.001); blocked {
			continue
		}

		// Lambertian cosine factor
		cosTheta := hit.Normal.Dot(lightDir)
		if cosTheta <= 0 {
			continue
		}

		// Inverse square falloff, divided by 4*pi for point source to hemisphere
		falloff := cosTheta / (4 * math.Pi * dist * dist)
		direct = direct.Add(light.Emission.Scale(falloff))
	}

	// Apply material reflectance to direct light
	scatter := mat.Scatter(dir, hit)
	directContrib := direct.Mul(scatter.Reflectance)

	// Indirect lighting (recursive bounce)
	var indirect spectrum.SPD
	if scatter.Scattered && depth > 1 {
		bounced := tr.traceWorld(hit.Point, scatter.OutDir, depth-1)
		indirect = bounced.Mul(scatter.Reflectance)
	}

	return emitted.Add(directContrib).Add(indirect)
}
