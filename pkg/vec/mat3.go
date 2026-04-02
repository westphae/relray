package vec

import "math"

// Mat3 is a 3x3 matrix stored as three row vectors.
type Mat3 [3]Vec3

// Identity returns the 3x3 identity matrix.
func Identity() Mat3 {
	return Mat3{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
}

// MulVec multiplies the matrix by a vector (M * v).
func (m Mat3) MulVec(v Vec3) Vec3 {
	return Vec3{
		X: m[0].Dot(v),
		Y: m[1].Dot(v),
		Z: m[2].Dot(v),
	}
}

// Transpose returns the transpose of the matrix.
// For rotation matrices, the transpose is the inverse.
func (m Mat3) Transpose() Mat3 {
	return Mat3{
		{m[0].X, m[1].X, m[2].X},
		{m[0].Y, m[1].Y, m[2].Y},
		{m[0].Z, m[1].Z, m[2].Z},
	}
}

// Mul returns the product of two matrices (a * b).
func (a Mat3) Mul(b Mat3) Mat3 {
	bt := b.Transpose()
	return Mat3{
		{a[0].Dot(bt[0]), a[0].Dot(bt[1]), a[0].Dot(bt[2])},
		{a[1].Dot(bt[0]), a[1].Dot(bt[1]), a[1].Dot(bt[2])},
		{a[2].Dot(bt[0]), a[2].Dot(bt[1]), a[2].Dot(bt[2])},
	}
}

// RotationX returns a rotation matrix around the X axis by angle radians.
func RotationX(angle float64) Mat3 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Mat3{
		{1, 0, 0},
		{0, c, -s},
		{0, s, c},
	}
}

// RotationY returns a rotation matrix around the Y axis by angle radians.
func RotationY(angle float64) Mat3 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Mat3{
		{c, 0, s},
		{0, 1, 0},
		{-s, 0, c},
	}
}

// RotationZ returns a rotation matrix around the Z axis by angle radians.
func RotationZ(angle float64) Mat3 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Mat3{
		{c, -s, 0},
		{s, c, 0},
		{0, 0, 1},
	}
}

// RotationFromEulerDeg constructs a rotation matrix from Euler angles in degrees.
// Applied as: R = Ry(yaw) * Rx(pitch) * Rz(roll).
func RotationFromEulerDeg(yaw, pitch, roll float64) Mat3 {
	y := yaw * math.Pi / 180
	p := pitch * math.Pi / 180
	r := roll * math.Pi / 180
	return RotationY(y).Mul(RotationX(p)).Mul(RotationZ(r))
}

// RotationFromAxisAngle constructs a rotation matrix using Rodrigues' formula.
// axis must be a unit vector, angle is in radians.
func RotationFromAxisAngle(axis Vec3, angle float64) Mat3 {
	c, s := math.Cos(angle), math.Sin(angle)
	t := 1 - c
	x, y, z := axis.X, axis.Y, axis.Z
	return Mat3{
		{t*x*x + c, t*x*y - s*z, t*x*z + s*y},
		{t*x*y + s*z, t*y*y + c, t*y*z - s*x},
		{t*x*z - s*y, t*y*z + s*x, t*z*z + c},
	}
}
