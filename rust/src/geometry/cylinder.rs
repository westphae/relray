use crate::vec::Vec3;
use super::{Hit, Shape};

pub struct Cylinder {
    pub radius: f64,
    pub height: f64,
}

impl Shape for Cylinder {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        let a = dir.x * dir.x + dir.z * dir.z;
        let b = 2.0 * (origin.x * dir.x + origin.z * dir.z);
        let cc = origin.x * origin.x + origin.z * origin.z - self.radius * self.radius;

        let mut best_t: f64 = -1.0;
        let mut best_normal = Vec3::default();
        let mut found = false;

        let disc = b * b - 4.0 * a * cc;
        if disc >= 0.0 && a > 1e-12 {
            let sqrt_disc = disc.sqrt();
            for t in [(-b - sqrt_disc) / (2.0 * a), (-b + sqrt_disc) / (2.0 * a)] {
                if t < t_min || t > t_max {
                    continue;
                }
                let y = origin.y + t * dir.y;
                if y >= 0.0 && y <= self.height {
                    if !found || t < best_t {
                        best_t = t;
                        let p = origin + dir * t;
                        best_normal = Vec3 { x: p.x, y: 0.0, z: p.z }.normalize();
                        found = true;
                    }
                    break;
                }
            }
        }

        // Test caps (Y=0 and Y=Height)
        if dir.y.abs() > 1e-12 {
            for cap_y in [0.0, self.height] {
                let t = (cap_y - origin.y) / dir.y;
                if t < t_min || t > t_max {
                    continue;
                }
                let p = origin + dir * t;
                if p.x * p.x + p.z * p.z <= self.radius * self.radius {
                    if !found || t < best_t {
                        best_t = t;
                        best_normal = if cap_y == 0.0 {
                            Vec3 { x: 0.0, y: -1.0, z: 0.0 }
                        } else {
                            Vec3 { x: 0.0, y: 1.0, z: 0.0 }
                        };
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
