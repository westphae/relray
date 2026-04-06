mod sphere;
mod plane;
mod box_shape;
mod cylinder;
mod cone;
mod disk;
mod triangle;
mod pyramid;
mod transform;

pub use sphere::Sphere;
pub use plane::Plane;
pub use box_shape::BoxShape;
pub use cylinder::Cylinder;
pub use cone::Cone;
pub use disk::Disk;
pub use triangle::Triangle;
pub use pyramid::Pyramid;
pub use transform::Transformed;

use crate::vec::Vec3;

#[derive(Clone, Debug, Default)]
pub struct Hit {
    pub t: f64,
    pub point: Vec3,
    pub normal: Vec3,
    pub front_face: bool,
    pub source_velocity: Vec3,
}

impl Hit {
    pub fn set_face_normal(&mut self, ray_dir: Vec3, outward_normal: Vec3) {
        self.front_face = ray_dir.dot(outward_normal) < 0.0;
        self.normal = if self.front_face {
            outward_normal
        } else {
            -outward_normal
        };
    }
}

pub trait Shape: Send + Sync {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit>;
}
