package vec

import "math"

// Vec3 represents a 3D vector.
type Vec3 struct{ X, Y, Z float64 }

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Neg() Vec3          { return Vec3{-a.X, -a.Y, -a.Z} }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) LengthSq() float64 { return a.Dot(a) }
func (a Vec3) Length() float64    { return math.Sqrt(a.LengthSq()) }

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l == 0 {
		return Vec3{}
	}
	return a.Scale(1.0 / l)
}

// Reflect returns the reflection of a about the normal n (assumed unit length).
func (a Vec3) Reflect(n Vec3) Vec3 {
	return a.Sub(n.Scale(2 * a.Dot(n)))
}
