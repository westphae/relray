package render

import (
	"math"
	"math/rand"

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
	Rng      *rand.Rand
}

// Trace traces a single pixel at normalized screen coords (u, v).
func (tr *Tracer) Trace(u, v float64) spectrum.SPD {
	dirObs := tr.Camera.RayDir(u, v)

	ab := lorentz.Aberrate(dirObs, tr.Camera.Beta)
	dirWorld := ab.Dir
	doppler := ab.Doppler

	spd := tr.traceWorld(tr.Camera.Position, dirWorld, tr.MaxDepth)

	spd = spd.Shift(1.0 / doppler)
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
		if tr.Scene.Sky != nil {
			return tr.Scene.Sky(dir)
		}
		return spectrum.SPD{}
	}

	emitted := mat.Emitted(hit)

	// Direct lighting from point lights
	var direct spectrum.SPD
	for i := range tr.Scene.Lights {
		light := &tr.Scene.Lights[i]
		toLight := light.Position.Sub(hit.Point)
		dist := toLight.Length()
		lightDir := toLight.Scale(1.0 / dist)

		if _, _, blocked := tr.Scene.Intersect(hit.Point, lightDir, 0.001, dist-0.001); blocked {
			continue
		}

		cosTheta := hit.Normal.Dot(lightDir)
		if cosTheta <= 0 {
			continue
		}

		falloff := cosTheta / (4 * math.Pi * dist * dist)
		direct = direct.Add(light.Emission.Scale(falloff))
	}

	scatter := mat.Scatter(dir, hit, tr.Rng)
	directContrib := direct.Mul(scatter.Reflectance)

	// Indirect lighting (recursive bounce)
	var indirect spectrum.SPD
	if scatter.Scattered && depth > 1 {
		bounced := tr.traceWorld(hit.Point, scatter.OutDir, depth-1)
		indirect = bounced.Mul(scatter.Reflectance)
	}

	return emitted.Add(directContrib).Add(indirect)
}
