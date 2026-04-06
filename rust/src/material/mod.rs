mod diffuse;
mod mirror;
mod glass;
mod checker;
mod checker_sphere;

pub use diffuse::Diffuse;
pub use mirror::Mirror;
pub use glass::Glass;
pub use checker::Checker;
pub use checker_sphere::CheckerSphere;

use crate::geometry::Hit;
use crate::spectrum::Spd;
use crate::vec::Vec3;
use rand::Rng;
use rand::rngs::SmallRng;

pub struct ScatterResult {
    pub scattered: bool,
    pub out_dir: Vec3,
    pub reflectance: Spd,
}

pub trait Material: Send + Sync {
    fn scatter(&self, in_dir: Vec3, hit: &Hit, rng: &mut SmallRng) -> ScatterResult;
    fn emitted(&self, hit: &Hit) -> Spd;
}

/// Returns a uniformly distributed random unit vector.
pub fn random_unit_vec(rng: &mut SmallRng) -> Vec3 {
    loop {
        let v = Vec3::new(
            rng.random_range(-1.0_f64..1.0),
            rng.random_range(-1.0_f64..1.0),
            rng.random_range(-1.0_f64..1.0),
        );
        let l2 = v.length_sq();
        if l2 > 1e-6 && l2 <= 1.0 {
            return v * (1.0 / l2.sqrt());
        }
    }
}
