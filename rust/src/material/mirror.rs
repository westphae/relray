use crate::geometry::Hit;
use crate::spectrum::Spd;
use crate::vec::Vec3;
use rand::rngs::SmallRng;

use super::{Material, ScatterResult};

/// Perfect specular reflector with spectral reflectance.
pub struct Mirror {
    pub reflectance: Spd,
}

impl Material for Mirror {
    fn scatter(&self, in_dir: Vec3, hit: &Hit, _rng: &mut SmallRng) -> ScatterResult {
        let reflected = in_dir.reflect(hit.normal);
        ScatterResult {
            scattered: true,
            out_dir: reflected.normalize(),
            reflectance: self.reflectance,
        }
    }

    fn emitted(&self, _hit: &Hit) -> Spd {
        Spd::default()
    }
}
