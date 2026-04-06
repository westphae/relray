use crate::vec::Vec3;
use super::{Hit, Shape};

pub struct Plane;

impl Shape for Plane {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        if dir.y.abs() < 1e-12 {
            return None;
        }
        let t = -origin.y / dir.y;
        if t < t_min || t > t_max {
            return None;
        }
        let pt = origin + dir * t;
        let mut h = Hit { t, point: pt, ..Default::default() };
        h.set_face_normal(dir, Vec3 { x: 0.0, y: 1.0, z: 0.0 });
        Some(h)
    }
}
