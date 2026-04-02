package material

import (
	"sif/gogs/eric/relray/pkg/geometry"
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// ScatterResult describes what happens when light hits a surface.
type ScatterResult struct {
	Scattered   bool
	OutDir      vec.Vec3     // scattered ray direction (for recursive tracing)
	Reflectance spectrum.SPD // spectral reflectance applied to incoming light
}

// Material determines how a surface interacts with light at each wavelength.
type Material interface {
	Scatter(inDir vec.Vec3, hit geometry.Hit) ScatterResult
	Emitted(hit geometry.Hit) spectrum.SPD
}
