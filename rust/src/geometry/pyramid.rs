use std::f64::consts::PI;
use std::sync::OnceLock;

use crate::vec::Vec3;
use super::triangle::Triangle;
use super::{Hit, Shape};

pub struct Pyramid {
    pub base_radius: f64,
    pub height: f64,
    pub sides: usize,
    faces: OnceLock<Vec<Triangle>>,
}

impl Pyramid {
    pub fn new(base_radius: f64, height: f64, sides: usize) -> Self {
        Self {
            base_radius,
            height,
            sides,
            faces: OnceLock::new(),
        }
    }

    fn build_faces(&self) -> Vec<Triangle> {
        let n = if self.sides < 3 { 3 } else { self.sides };
        let apex = Vec3 { x: 0.0, y: self.height, z: 0.0 };

        let verts: Vec<Vec3> = (0..n)
            .map(|i| {
                let angle = 2.0 * PI * (i as f64) / (n as f64);
                Vec3 {
                    x: self.base_radius * angle.cos(),
                    y: 0.0,
                    z: self.base_radius * angle.sin(),
                }
            })
            .collect();

        let mut faces = Vec::with_capacity(2 * n);

        // Side faces
        for i in 0..n {
            let j = (i + 1) % n;
            faces.push(Triangle { v0: apex, v1: verts[i], v2: verts[j] });
        }

        // Base faces (fan triangulation, wound opposite to sides so normal points down)
        let base_center = Vec3::default();
        for i in 0..n {
            let j = (i + 1) % n;
            faces.push(Triangle { v0: base_center, v1: verts[j], v2: verts[i] });
        }

        faces
    }

    fn faces(&self) -> &[Triangle] {
        self.faces.get_or_init(|| self.build_faces())
    }
}

impl Shape for Pyramid {
    fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<Hit> {
        let mut closest: Option<Hit> = None;
        let mut best = t_max;

        for tri in self.faces() {
            if let Some(h) = tri.intersect_tri(origin, dir, t_min, best) {
                best = h.t;
                closest = Some(h);
            }
        }
        closest
    }
}
