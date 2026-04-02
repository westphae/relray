package scene

import (
	"sif/gogs/eric/relray/pkg/spectrum"
	"sif/gogs/eric/relray/pkg/vec"
)

// Light is a point light source with a spectral emission profile.
type Light struct {
	Position vec.Vec3
	Emission spectrum.SPD
}
