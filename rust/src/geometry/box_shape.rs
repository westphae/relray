use crate::vec::Vec3;
use super::{Hit, Shape};

pub struct BoxShape {
    pub size: Vec3,
}

impl Shape for BoxShape {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        let half_x = self.size.x / 2.0;
        let half_y = self.size.y / 2.0;
        let half_z = self.size.z / 2.0;

        let inv_d = Vec3 {
            x: 1.0 / dir.x,
            y: 1.0 / dir.y,
            z: 1.0 / dir.z,
        };

        let (mut t0x, mut t1x) = ((-half_x - origin.x) * inv_d.x, (half_x - origin.x) * inv_d.x);
        if inv_d.x < 0.0 {
            std::mem::swap(&mut t0x, &mut t1x);
        }

        let (mut t0y, mut t1y) = ((-half_y - origin.y) * inv_d.y, (half_y - origin.y) * inv_d.y);
        if inv_d.y < 0.0 {
            std::mem::swap(&mut t0y, &mut t1y);
        }

        let (mut t0z, mut t1z) = ((-half_z - origin.z) * inv_d.z, (half_z - origin.z) * inv_d.z);
        if inv_d.z < 0.0 {
            std::mem::swap(&mut t0z, &mut t1z);
        }

        let t_near = t0x.max(t0y.max(t0z));
        let t_far = t1x.min(t1y.min(t1z));

        if t_near > t_far || t_far < t_min || t_near > t_max {
            return None;
        }

        let mut t = t_near;
        if t < t_min {
            t = t_far;
            if t > t_max {
                return None;
            }
        }

        let p = origin + dir * t;

        const BIAS: f64 = 1e-6;
        let normal = if (p.x + half_x).abs() < BIAS {
            Vec3 { x: -1.0, y: 0.0, z: 0.0 }
        } else if (p.x - half_x).abs() < BIAS {
            Vec3 { x: 1.0, y: 0.0, z: 0.0 }
        } else if (p.y + half_y).abs() < BIAS {
            Vec3 { x: 0.0, y: -1.0, z: 0.0 }
        } else if (p.y - half_y).abs() < BIAS {
            Vec3 { x: 0.0, y: 1.0, z: 0.0 }
        } else if (p.z + half_z).abs() < BIAS {
            Vec3 { x: 0.0, y: 0.0, z: -1.0 }
        } else {
            Vec3 { x: 0.0, y: 0.0, z: 1.0 }
        };

        let mut h = Hit { t, point: p, ..Default::default() };
        h.set_face_normal(dir, normal);
        Some(h)
    }
}
