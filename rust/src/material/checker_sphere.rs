use std::f64::consts::PI;

use crate::geometry::Hit;
use crate::spectrum::Spd;
use crate::vec::Vec3;
use rand::rngs::SmallRng;

use super::{random_unit_vec, Material, ScatterResult};

/// Lambertian diffuse material with a checkerboard pattern mapped onto a
/// sphere using latitude/longitude coordinates derived from the surface
/// normal. `num_squares` controls how many checker divisions there are
/// around the equator (and half that many pole-to-pole).
pub struct CheckerSphere {
    pub even: Spd,
    pub odd: Spd,
    /// Divisions around equator (default 8).
    pub num_squares: i32,
}

impl CheckerSphere {
    fn reflectance_at(&self, normal: Vec3) -> Spd {
        let n = if self.num_squares <= 0 { 8 } else { self.num_squares };

        // Latitude: -PI/2 to PI/2, longitude: -PI to PI
        let lat = normal.y.clamp(-1.0, 1.0).asin();
        let lon = normal.z.atan2(normal.x);

        // Map to grid squares
        let lat_div = (lat * f64::from(n) / PI).floor() as i64;
        let lon_div = (lon * f64::from(n) / PI).floor() as i64;

        if (lat_div + lon_div) % 2 == 0 {
            self.even
        } else {
            self.odd
        }
    }
}

impl Material for CheckerSphere {
    fn scatter(&self, _in_dir: Vec3, hit: &Hit, rng: &mut SmallRng) -> ScatterResult {
        let mut scattered = hit.normal + random_unit_vec(rng);
        if scattered.length_sq() < 1e-12 {
            scattered = hit.normal;
        }
        ScatterResult {
            scattered: true,
            out_dir: scattered.normalize(),
            reflectance: self.reflectance_at(hit.normal),
        }
    }

    fn emitted(&self, _hit: &Hit) -> Spd {
        Spd::default()
    }
}
