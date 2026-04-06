use crate::vec::{Mat3, Vec3};
use super::{Hit, Shape};

pub struct Transformed {
    pub shape: Box<dyn Shape>,
    pub position: Vec3,
    pub rotation: Mat3,
    pub inv_rot: Mat3,
}

impl Transformed {
    pub fn new(shape: Box<dyn Shape>, position: Vec3, rotation: Mat3) -> Self {
        Self {
            shape,
            position,
            rotation: rotation,
            inv_rot: rotation.transpose(),
        }
    }
}

impl Shape for Transformed {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        let local_origin = self.inv_rot.mul_vec(origin - self.position);
        let local_dir = self.inv_rot.mul_vec(dir);

        let mut h = self.shape.intersect(local_origin, local_dir, t_min, t_max)?;

        h.point = self.rotation.mul_vec(h.point) + self.position;
        h.normal = self.rotation.mul_vec(h.normal).normalize();
        Some(h)
    }
}
