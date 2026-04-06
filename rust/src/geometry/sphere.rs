use crate::vec::Vec3;
use super::{Hit, Shape};

pub struct Sphere {
    pub radius: f64,
}

impl Shape for Sphere {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        let a = dir.length_sq();
        let half_b = origin.dot(dir);
        let c = origin.length_sq() - self.radius * self.radius;
        let disc = half_b * half_b - a * c;
        if disc < 0.0 {
            return None;
        }

        let sqrt_disc = disc.sqrt();
        let mut t = (-half_b - sqrt_disc) / a;
        if t < t_min || t > t_max {
            t = (-half_b + sqrt_disc) / a;
            if t < t_min || t > t_max {
                return None;
            }
        }

        let p = origin + dir * t;
        let outward = p * (1.0 / self.radius);
        let mut h = Hit { t, point: p, ..Default::default() };
        h.set_face_normal(dir, outward);
        Some(h)
    }
}
