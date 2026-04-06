use crate::geometry::Hit;
use crate::spectrum::Spd;
use crate::vec::Vec3;
use rand::rngs::SmallRng;

use super::{random_unit_vec, Material, ScatterResult};

/// Lambertian diffuse material with a checkerboard pattern alternating
/// between two spectral reflectances. The pattern is computed in the
/// tangent plane of the surface (perpendicular to the hit normal), so it
/// produces a proper checkerboard on every flat face of any shape.
pub struct Checker {
    pub even: Spd,
    pub odd: Spd,
    /// Size of each checker square in world units.
    pub scale: f64,
}

impl Checker {
    fn reflectance_at(&self, p: Vec3, n: Vec3) -> Spd {
        // Build a tangent frame from the surface normal.
        let reference = if n.x.abs() > 0.9 {
            Vec3 { x: 0.0, y: 1.0, z: 0.0 }
        } else {
            Vec3 { x: 1.0, y: 0.0, z: 0.0 }
        };
        let t1 = n.cross(reference).normalize();
        let t2 = n.cross(t1);

        let inv = 1.0 / self.scale;
        let iu = (p.dot(t1) * inv).floor() as i64;
        let iv = (p.dot(t2) * inv).floor() as i64;
        if (iu + iv) % 2 == 0 {
            self.even
        } else {
            self.odd
        }
    }
}

impl Material for Checker {
    fn scatter(&self, _in_dir: Vec3, hit: &Hit, rng: &mut SmallRng) -> ScatterResult {
        let mut scattered = hit.normal + random_unit_vec(rng);
        if scattered.length_sq() < 1e-12 {
            scattered = hit.normal;
        }
        ScatterResult {
            scattered: true,
            out_dir: scattered.normalize(),
            reflectance: self.reflectance_at(hit.point, hit.normal),
        }
    }

    fn emitted(&self, _hit: &Hit) -> Spd {
        Spd::default()
    }
}
