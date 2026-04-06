package scenefile

// YAML intermediate types for scene file deserialization.

// SceneFile is the top-level YAML structure.
type SceneFile struct {
	Name           string          `yaml:"name"`
	Camera         *CameraSpec     `yaml:"camera"`
	Sky            *SkySpec        `yaml:"sky"`
	Lights         []LightSpec     `yaml:"lights"`
	Objects        []ObjectSpec    `yaml:"objects"`
	MovingObjects  []MovingObjSpec `yaml:"moving_objects"`
}

// CameraSpec defines the camera.
type CameraSpec struct {
	Position [3]float64  `yaml:"position"`
	LookAt   [3]float64  `yaml:"look_at"`
	Up       [3]float64  `yaml:"up"`
	VFOV     float64     `yaml:"vfov"`
	Velocity *[3]float64 `yaml:"velocity,omitempty"` // world-frame velocity as fraction of c
}

// SkySpec defines the sky/environment.
type SkySpec struct {
	Type     string   `yaml:"type"`              // "gradient", "uniform", "none"
	Top      *SPDSpec `yaml:"top,omitempty"`      // for gradient
	Bottom   *SPDSpec `yaml:"bottom,omitempty"`   // for gradient
	Emission *SPDSpec `yaml:"emission,omitempty"` // for uniform
}

// LightSpec defines a point light.
type LightSpec struct {
	Position [3]float64 `yaml:"position"`
	Emission SPDSpec    `yaml:"emission"`
}

// ObjectSpec defines a static object.
type ObjectSpec struct {
	Shape    ShapeSpec    `yaml:"shape"`
	Material MaterialSpec `yaml:"material"`
}

// MovingObjSpec defines a moving object.
type MovingObjSpec struct {
	Shape      ShapeSpec      `yaml:"shape"`
	Material   MaterialSpec   `yaml:"material"`
	Trajectory TrajectorySpec `yaml:"trajectory"`
}

// ShapeSpec is a tagged union for shapes.
// Exactly one shape field should be set.
// Position and Rotation are optional transforms applied to any shape.
type ShapeSpec struct {
	Sphere   *SphereSpec   `yaml:"sphere,omitempty"`
	Plane    *PlaneSpec    `yaml:"plane,omitempty"`
	Box      *BoxSpec      `yaml:"box,omitempty"`
	Cylinder *CylinderSpec `yaml:"cylinder,omitempty"`
	Cone     *ConeSpec     `yaml:"cone,omitempty"`
	Disk     *DiskSpec     `yaml:"disk,omitempty"`
	Triangle *TriangleSpec `yaml:"triangle,omitempty"`
	Pyramid  *PyramidSpec  `yaml:"pyramid,omitempty"`

	// Optional transform (applied to any shape)
	Position *[3]float64 `yaml:"position,omitempty"` // translation
	Rotation *[3]float64 `yaml:"rotation,omitempty"` // euler angles in degrees: yaw, pitch, roll
}

type SphereSpec struct {
	Radius float64 `yaml:"radius"`
}

type PlaneSpec struct{} // XZ plane at origin, normal +Y; use position/rotation to place

type BoxSpec struct {
	Size [3]float64 `yaml:"size"` // [width, height, depth], centered at origin
}

type CylinderSpec struct {
	Radius float64 `yaml:"radius"`
	Height float64 `yaml:"height"`
}

type ConeSpec struct {
	Radius float64 `yaml:"radius"`
	Height float64 `yaml:"height"`
}

type DiskSpec struct {
	Radius float64 `yaml:"radius"` // on XZ plane at origin, normal +Y
}

type TriangleSpec struct {
	V0 [3]float64 `yaml:"v0"`
	V1 [3]float64 `yaml:"v1"`
	V2 [3]float64 `yaml:"v2"`
}

type PyramidSpec struct {
	BaseRadius float64 `yaml:"base_radius"`
	Height     float64 `yaml:"height"`
	Sides      int     `yaml:"sides"`
}

// MaterialSpec is a tagged union for materials.
// Exactly one field should be set.
type MaterialSpec struct {
	Diffuse       *DiffuseMatSpec       `yaml:"diffuse,omitempty"`
	Mirror        *MirrorMatSpec        `yaml:"mirror,omitempty"`
	Glass         *GlassMatSpec         `yaml:"glass,omitempty"`
	Checker       *CheckerMatSpec       `yaml:"checker,omitempty"`
	CheckerSphere *CheckerSphereMatSpec `yaml:"checker_sphere,omitempty"`
}

type DiffuseMatSpec struct {
	SPDSpec `yaml:",inline"` // allows { rgb: [...] } directly
}

type MirrorMatSpec struct {
	SPDSpec `yaml:",inline"`
}

type GlassMatSpec struct {
	IOR  float64 `yaml:"ior"`
	Tint SPDSpec `yaml:"tint"`
}

type CheckerMatSpec struct {
	Even  SPDSpec `yaml:"even"`
	Odd   SPDSpec `yaml:"odd"`
	Scale float64 `yaml:"scale"`
}

type CheckerSphereMatSpec struct {
	Even       SPDSpec `yaml:"even"`
	Odd        SPDSpec `yaml:"odd"`
	NumSquares int     `yaml:"num_squares"`
}

// TrajectorySpec is a tagged union for trajectories.
type TrajectorySpec struct {
	Static  *StaticTrajSpec  `yaml:"static,omitempty"`
	Linear  *LinearTrajSpec  `yaml:"linear,omitempty"`
	Orbit   *OrbitTrajSpec   `yaml:"orbit,omitempty"`
}

type StaticTrajSpec struct {
	Position [3]float64 `yaml:"position"`
}

type LinearTrajSpec struct {
	Start    [3]float64 `yaml:"start"`
	Velocity [3]float64 `yaml:"velocity"`
}

type OrbitTrajSpec struct {
	Center [3]float64 `yaml:"center"`
	Radius float64    `yaml:"radius"`
	Period float64    `yaml:"period"`
	Axis   string     `yaml:"axis"` // "x", "y", or "z"
}

// SPDSpec is a tagged union for spectral power distributions.
// Exactly one field should be set.
type SPDSpec struct {
	RGB           *[3]float64        `yaml:"rgb,omitempty"`
	Blackbody     *BlackbodySpec     `yaml:"blackbody,omitempty"`
	Constant      *float64           `yaml:"constant,omitempty"`
	D65           *float64           `yaml:"d65,omitempty"`
	Monochromatic *MonochromaticSpec `yaml:"monochromatic,omitempty"`
	Reflectance   *[][2]float64      `yaml:"reflectance,omitempty"` // [[wavelength_nm, value], ...]
}

type BlackbodySpec struct {
	Temp      float64 `yaml:"temp"`
	Luminance float64 `yaml:"luminance"`
}

type MonochromaticSpec struct {
	Wavelength float64 `yaml:"wavelength"`
	Power      float64 `yaml:"power"`
}
