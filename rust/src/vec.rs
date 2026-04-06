use std::ops;

/// A 3D vector with f64 components.
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct Vec3 {
    pub x: f64,
    pub y: f64,
    pub z: f64,
}

impl Vec3 {
    pub fn new(x: f64, y: f64, z: f64) -> Self {
        Self { x, y, z }
    }

    pub fn dot(self, b: Self) -> f64 {
        self.x * b.x + self.y * b.y + self.z * b.z
    }

    pub fn cross(self, b: Self) -> Self {
        Self {
            x: self.y * b.z - self.z * b.y,
            y: self.z * b.x - self.x * b.z,
            z: self.x * b.y - self.y * b.x,
        }
    }

    pub fn length_sq(self) -> f64 {
        self.dot(self)
    }

    pub fn length(self) -> f64 {
        self.length_sq().sqrt()
    }

    pub fn normalize(self) -> Self {
        let l = self.length();
        if l == 0.0 {
            Self::default()
        } else {
            self * (1.0 / l)
        }
    }

    /// Reflect self about the normal n (assumed unit length).
    pub fn reflect(self, n: Self) -> Self {
        self - n * (2.0 * self.dot(n))
    }
}

// Vec3 + Vec3
impl ops::Add for Vec3 {
    type Output = Self;
    fn add(self, rhs: Self) -> Self {
        Self {
            x: self.x + rhs.x,
            y: self.y + rhs.y,
            z: self.z + rhs.z,
        }
    }
}

// Vec3 - Vec3
impl ops::Sub for Vec3 {
    type Output = Self;
    fn sub(self, rhs: Self) -> Self {
        Self {
            x: self.x - rhs.x,
            y: self.y - rhs.y,
            z: self.z - rhs.z,
        }
    }
}

// -Vec3
impl ops::Neg for Vec3 {
    type Output = Self;
    fn neg(self) -> Self {
        Self {
            x: -self.x,
            y: -self.y,
            z: -self.z,
        }
    }
}

// Vec3 * f64
impl ops::Mul<f64> for Vec3 {
    type Output = Self;
    fn mul(self, rhs: f64) -> Self {
        Self {
            x: self.x * rhs,
            y: self.y * rhs,
            z: self.z * rhs,
        }
    }
}

// f64 * Vec3
impl ops::Mul<Vec3> for f64 {
    type Output = Vec3;
    fn mul(self, rhs: Vec3) -> Vec3 {
        Vec3 {
            x: self * rhs.x,
            y: self * rhs.y,
            z: self * rhs.z,
        }
    }
}

// Vec3 / f64
impl ops::Div<f64> for Vec3 {
    type Output = Self;
    fn div(self, rhs: f64) -> Self {
        Self {
            x: self.x / rhs,
            y: self.y / rhs,
            z: self.z / rhs,
        }
    }
}

// Vec3 += Vec3
impl ops::AddAssign for Vec3 {
    fn add_assign(&mut self, rhs: Self) {
        self.x += rhs.x;
        self.y += rhs.y;
        self.z += rhs.z;
    }
}

// Vec3 -= Vec3
impl ops::SubAssign for Vec3 {
    fn sub_assign(&mut self, rhs: Self) {
        self.x -= rhs.x;
        self.y -= rhs.y;
        self.z -= rhs.z;
    }
}

// Vec3 *= f64
impl ops::MulAssign<f64> for Vec3 {
    fn mul_assign(&mut self, rhs: f64) {
        self.x *= rhs;
        self.y *= rhs;
        self.z *= rhs;
    }
}

/// A 3x3 matrix stored as three row vectors.
#[derive(Clone, Copy, Debug)]
pub struct Mat3(pub [Vec3; 3]);

impl Mat3 {
    pub fn identity() -> Self {
        Mat3([
            Vec3::new(1.0, 0.0, 0.0),
            Vec3::new(0.0, 1.0, 0.0),
            Vec3::new(0.0, 0.0, 1.0),
        ])
    }

    /// Multiply matrix by vector (M * v).
    pub fn mul_vec(self, v: Vec3) -> Vec3 {
        Vec3::new(self.0[0].dot(v), self.0[1].dot(v), self.0[2].dot(v))
    }

    /// Return the transpose of the matrix.
    pub fn transpose(self) -> Self {
        Mat3([
            Vec3::new(self.0[0].x, self.0[1].x, self.0[2].x),
            Vec3::new(self.0[0].y, self.0[1].y, self.0[2].y),
            Vec3::new(self.0[0].z, self.0[1].z, self.0[2].z),
        ])
    }

    /// Multiply two matrices (self * b).
    pub fn mul_mat(self, b: Self) -> Self {
        let bt = b.transpose();
        Mat3([
            Vec3::new(
                self.0[0].dot(bt.0[0]),
                self.0[0].dot(bt.0[1]),
                self.0[0].dot(bt.0[2]),
            ),
            Vec3::new(
                self.0[1].dot(bt.0[0]),
                self.0[1].dot(bt.0[1]),
                self.0[1].dot(bt.0[2]),
            ),
            Vec3::new(
                self.0[2].dot(bt.0[0]),
                self.0[2].dot(bt.0[1]),
                self.0[2].dot(bt.0[2]),
            ),
        ])
    }

    /// Rotation matrix around the X axis by `angle` radians.
    pub fn rotation_x(angle: f64) -> Self {
        let (s, c) = angle.sin_cos();
        Mat3([
            Vec3::new(1.0, 0.0, 0.0),
            Vec3::new(0.0, c, -s),
            Vec3::new(0.0, s, c),
        ])
    }

    /// Rotation matrix around the Y axis by `angle` radians.
    pub fn rotation_y(angle: f64) -> Self {
        let (s, c) = angle.sin_cos();
        Mat3([
            Vec3::new(c, 0.0, s),
            Vec3::new(0.0, 1.0, 0.0),
            Vec3::new(-s, 0.0, c),
        ])
    }

    /// Rotation matrix around the Z axis by `angle` radians.
    pub fn rotation_z(angle: f64) -> Self {
        let (s, c) = angle.sin_cos();
        Mat3([
            Vec3::new(c, -s, 0.0),
            Vec3::new(s, c, 0.0),
            Vec3::new(0.0, 0.0, 1.0),
        ])
    }

    /// Construct a rotation from Euler angles in degrees.
    /// Applied as: R = Ry(yaw) * Rx(pitch) * Rz(roll).
    pub fn from_euler_deg(yaw: f64, pitch: f64, roll: f64) -> Self {
        Self::rotation_y(yaw.to_radians())
            .mul_mat(Self::rotation_x(pitch.to_radians()))
            .mul_mat(Self::rotation_z(roll.to_radians()))
    }

    /// Construct a rotation using Rodrigues' formula.
    /// `axis` must be a unit vector, `angle` is in radians.
    #[allow(dead_code)]
    pub fn from_axis_angle(axis: Vec3, angle: f64) -> Self {
        let (s, c) = angle.sin_cos();
        let t = 1.0 - c;
        let x = axis.x;
        let y = axis.y;
        let z = axis.z;
        Mat3([
            Vec3::new(t * x * x + c, t * x * y - s * z, t * x * z + s * y),
            Vec3::new(t * x * y + s * z, t * y * y + c, t * y * z - s * x),
            Vec3::new(t * x * z - s * y, t * y * z + s * x, t * z * z + c),
        ])
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const EPS: f64 = 1e-12;

    fn assert_close(label: &str, got: f64, want: f64) {
        assert!(
            (got - want).abs() < EPS,
            "{label}: got {got}, want {want}"
        );
    }

    fn assert_vec(label: &str, got: Vec3, want: Vec3) {
        assert_close(&format!("{label}.x"), got.x, want.x);
        assert_close(&format!("{label}.y"), got.y, want.y);
        assert_close(&format!("{label}.z"), got.z, want.z);
    }

    #[test]
    fn test_basic_ops() {
        let a = Vec3::new(1.0, 2.0, 3.0);
        let b = Vec3::new(4.0, 5.0, 6.0);
        assert_vec("Add", a + b, Vec3::new(5.0, 7.0, 9.0));
        assert_vec("Sub", a - b, Vec3::new(-3.0, -3.0, -3.0));
        assert_vec("Scale", a * 2.0, Vec3::new(2.0, 4.0, 6.0));
        assert_close("Dot", a.dot(b), 32.0);
        assert_vec("Cross", a.cross(b), Vec3::new(-3.0, 6.0, -3.0));
        assert_vec("Neg", -a, Vec3::new(-1.0, -2.0, -3.0));
    }

    #[test]
    fn test_length() {
        let a = Vec3::new(3.0, 4.0, 0.0);
        assert_close("Length", a.length(), 5.0);
        assert_close("LengthSq", a.length_sq(), 25.0);
    }

    #[test]
    fn test_normalize() {
        let a = Vec3::new(0.0, 3.0, 4.0);
        let n = a.normalize();
        assert_close("normalized length", n.length(), 1.0);
        assert_vec("Normalize", n, Vec3::new(0.0, 0.6, 0.8));

        let zero = Vec3::default();
        assert_vec("zero normalize", zero.normalize(), Vec3::default());
    }

    #[test]
    fn test_reflect() {
        let d = Vec3::new(1.0, -1.0, 0.0).normalize();
        let n = Vec3::new(0.0, 1.0, 0.0);
        let r = d.reflect(n);
        let want = Vec3::new(1.0, 1.0, 0.0).normalize();
        assert_vec("Reflect", r, want);
    }

    #[test]
    fn test_identity_mul_vec() {
        let v = Vec3::new(1.0, 2.0, 3.0);
        let got = Mat3::identity().mul_vec(v);
        assert_vec("identity", got, v);
    }

    #[test]
    fn test_rotation_y_round_trip() {
        let r = Mat3::rotation_y(std::f64::consts::FRAC_PI_4);
        let v = Vec3::new(1.0, 0.0, 0.0);
        let rotated = r.mul_vec(v);
        let recovered = r.transpose().mul_vec(rotated);
        assert_vec("round-trip", recovered, v);
    }

    #[test]
    fn test_rotation_y_90() {
        let r = Mat3::rotation_y(std::f64::consts::FRAC_PI_2);
        let v = Vec3::new(1.0, 0.0, 0.0);
        let got = r.mul_vec(v);
        assert_vec("Ry(90)*X", got, Vec3::new(0.0, 0.0, -1.0));
    }

    #[test]
    fn test_rotation_x_90() {
        let r = Mat3::rotation_x(std::f64::consts::FRAC_PI_2);
        let v = Vec3::new(0.0, 1.0, 0.0);
        let got = r.mul_vec(v);
        assert_vec("Rx(90)*Y", got, Vec3::new(0.0, 0.0, 1.0));
    }

    #[test]
    fn test_rotation_from_euler_deg_identity() {
        let r = Mat3::from_euler_deg(0.0, 0.0, 0.0);
        let v = Vec3::new(1.0, 2.0, 3.0);
        let got = r.mul_vec(v);
        assert_vec("euler(0,0,0)", got, v);
    }

    #[test]
    fn test_rotation_from_axis_angle() {
        let r = Mat3::from_axis_angle(Vec3::new(0.0, 1.0, 0.0), std::f64::consts::FRAC_PI_2);
        let got = r.mul_vec(Vec3::new(1.0, 0.0, 0.0));
        assert_vec("axis-angle Y 90", got, Vec3::new(0.0, 0.0, -1.0));
    }

    #[test]
    fn test_rotation_orthogonal() {
        let r = Mat3::from_euler_deg(30.0, 45.0, 60.0);
        let product = r.mul_mat(r.transpose());
        let id = Mat3::identity();
        for i in 0..3 {
            assert_vec(
                "orthogonal row",
                product.0[i],
                id.0[i],
            );
        }
    }
}
