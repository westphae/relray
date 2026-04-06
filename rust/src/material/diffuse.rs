use crate::geometry::Hit;
use crate::spectrum::Spd;
use crate::vec::Vec3;
use rand::rngs::SmallRng;

use super::{random_unit_vec, Material, ScatterResult};

/// Lambertian diffuse material with spectral reflectance.
pub struct Diffuse {
    pub reflectance: Spd,
}

impl Material for Diffuse {
    fn scatter(&self, _in_dir: Vec3, hit: &Hit, rng: &mut SmallRng) -> ScatterResult {
        let mut scattered = hit.normal + random_unit_vec(rng);
        if scattered.length_sq() < 1e-12 {
            scattered = hit.normal;
        }
        ScatterResult {
            scattered: true,
            out_dir: scattered.normalize(),
            reflectance: self.reflectance,
        }
    }

    fn emitted(&self, _hit: &Hit) -> Spd {
        Spd::default()
    }
}
