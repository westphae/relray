use crate::vec::Vec3;
use super::{Hit, Shape};

pub struct Cone {
    pub radius: f64,
    pub height: f64,
}

impl Shape for Cone {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        let k = self.radius / self.height;
        let k2 = k * k;

        let hy = self.height - origin.y;
        let a = dir.x * dir.x + dir.z * dir.z - k2 * dir.y * dir.y;
        let b = 2.0 * (origin.x * dir.x + origin.z * dir.z) + 2.0 * k2 * hy * dir.y;
        let cc = origin.x * origin.x + origin.z * origin.z - k2 * hy * hy;

        let mut best_t: f64 = 0.0;
        let mut best_normal = Vec3::default();
        let mut found = false;

        let disc = b * b - 4.0 * a * cc;
        if disc >= 0.0 && a.abs() > 1e-12 {
            let sqrt_disc = disc.sqrt();
            for t in [(-b - sqrt_disc) / (2.0 * a), (-b + sqrt_disc) / (2.0 * a)] {
                if t < t_min || t > t_max {
                    continue;
                }
                let p = origin + dir * t;
                if p.y >= 0.0 && p.y <= self.height {
                    if !found || t < best_t {
                        best_t = t;
                        let r = (p.x * p.x + p.z * p.z).sqrt();
                        best_normal = if r > 1e-12 {
                            Vec3 {
                                x: p.x / r,
                                y: k,
                                z: p.z / r,
                            }
                            .normalize()
                        } else {
                            Vec3 { x: 0.0, y: 1.0, z: 0.0 }
                        };
                        found = true;
                    }
                    break;
                }
            }
        }

        // Test base cap (Y=0)
        if dir.y.abs() > 1e-12 {
            let t = -origin.y / dir.y;
            if t >= t_min && t <= t_max {
                let p = origin + dir * t;
                if p.x * p.x + p.z * p.z <= self.radius * self.radius {
                    if !found || t < best_t {
                        best_t = t;
                        best_normal = Vec3 { x: 0.0, y: -1.0, z: 0.0 };
                        found = true;
                    }
                }
            }
        }

        if !found {
            return None;
        }
        let p = origin + dir * best_t;
        let mut h = Hit { t: best_t, point: p, ..Default::default() };
        h.set_face_normal(dir, best_normal);
        Some(h)
    }
}
