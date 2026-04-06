use crate::vec::Vec3;
use super::{Hit, Shape};

pub struct Triangle {
    pub v0: Vec3,
    pub v1: Vec3,
    pub v2: Vec3,
}

impl Triangle {
    pub fn intersect_tri(
        &self,
        origin: Vec3,
        dir: Vec3,
        t_min: f64,
        t_max: f64,
    ) -> Option<Hit> {
        let edge1 = self.v1 - self.v0;
        let edge2 = self.v2 - self.v0;
        let h = dir.cross(edge2);
        let a = edge1.dot(h);

        if a.abs() < 1e-12 {
            return None;
        }

        let f = 1.0 / a;
        let s = origin - self.v0;
        let u = f * s.dot(h);
        if u < 0.0 || u > 1.0 {
            return None;
        }

        let q = s.cross(edge1);
        let v = f * dir.dot(q);
        if v < 0.0 || u + v > 1.0 {
            return None;
        }

        let t = f * edge2.dot(q);
        if t < t_min || t > t_max {
            return None;
        }

        let p = origin + dir * t;
        let normal = edge1.cross(edge2).normalize();
        let mut hit = Hit { t, point: p, ..Default::default() };
        hit.set_face_normal(dir, normal);
        Some(hit)
    }
}

impl Shape for Triangle {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        self.intersect_tri(origin, dir, t_min, t_max)
    }
}
