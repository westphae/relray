use crate::geometry::Hit;
use crate::spectrum::Spd;
use crate::vec::Vec3;
use rand::rngs::SmallRng;

use super::{random_unit_vec, Material, ScatterResult};

/// Lambertian diffuse material with a checkerboard pattern alternating
/// between two spectral reflectances. The pattern is defined in the XZ
/// plane (horizontal), with configurable scale.
pub struct Checker {
    pub even: Spd,
    pub odd: Spd,
    /// Size of each checker square in world units.
    pub scale: f64,
}

impl Checker {
    fn reflectance_at(&self, p: Vec3) -> Spd {
        let inv = 1.0 / self.scale;
        let ix = (p.x * inv).floor() as i64;
        let iz = (p.z * inv).floor() as i64;
        if (ix + iz) % 2 == 0 {
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
            reflectance: self.reflectance_at(hit.point),
        }
    }

    fn emitted(&self, _hit: &Hit) -> Spd {
        Spd::default()
    }
}
