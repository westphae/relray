use crate::geometry::Hit;
use crate::spectrum::Spd;
use crate::vec::Vec3;
use rand::Rng;
use rand::rngs::SmallRng;

use super::{Material, ScatterResult};

/// Dielectric material with refraction and Fresnel reflection.
/// Uses Schlick's approximation for the Fresnel term.
pub struct Glass {
    /// Index of refraction (e.g., 1.5 for typical glass).
    pub ior: f64,
    /// Spectral transmittance of the glass (1.0 = perfectly clear).
    /// Values < 1 at certain wavelengths create colored glass.
    pub tint: Spd,
}

impl Material for Glass {
    fn scatter(&self, in_dir: Vec3, hit: &Hit, rng: &mut SmallRng) -> ScatterResult {
        // Determine if we're entering or exiting the glass
        let eta_ratio = if hit.front_face {
            1.0 / self.ior // air -> glass
        } else {
            self.ior // glass -> air
        };

        let unit_dir = in_dir.normalize();
        let cos_i = (-unit_dir.dot(hit.normal)).min(1.0);
        let sin2_t = eta_ratio * eta_ratio * (1.0 - cos_i * cos_i);

        // Total internal reflection check
        if sin2_t > 1.0 {
            let reflected = unit_dir.reflect(hit.normal);
            return ScatterResult {
                scattered: true,
                out_dir: reflected.normalize(),
                reflectance: self.tint,
            };
        }

        // Schlick's approximation for Fresnel reflectance
        let reflectance = schlick(cos_i, eta_ratio);

        // Probabilistically choose reflection vs refraction
        if rng.random_range(0.0_f64..1.0) < reflectance {
            let reflected = unit_dir.reflect(hit.normal);
            return ScatterResult {
                scattered: true,
                out_dir: reflected.normalize(),
                reflectance: self.tint,
            };
        }

        // Refract
        let refracted = refract(unit_dir, hit.normal, eta_ratio);
        ScatterResult {
            scattered: true,
            out_dir: refracted.normalize(),
            reflectance: self.tint,
        }
    }

    fn emitted(&self, _hit: &Hit) -> Spd {
        Spd::default()
    }
}

/// Schlick's approximation to the Fresnel reflectance.
fn schlick(cosine: f64, eta_ratio: f64) -> f64 {
    let r0 = (1.0 - eta_ratio) / (1.0 + eta_ratio);
    let r0 = r0 * r0;
    r0 + (1.0 - r0) * (1.0 - cosine).powi(5)
}

/// Computes the refracted direction using Snell's law.
fn refract(uv: Vec3, n: Vec3, eta_ratio: f64) -> Vec3 {
    let cos_theta = (-uv.dot(n)).min(1.0);
    let r_perp = (uv + n * cos_theta) * eta_ratio;
    let r_parallel = n * -(1.0 - r_perp.length_sq()).abs().sqrt();
    r_perp + r_parallel
}
