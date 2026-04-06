use crate::geometry::{Hit, Shape};
use crate::material::Material;
use crate::retarded::{self, Trajectory};
use crate::spectrum::Spd;
use crate::vec::Vec3;

pub struct Light {
    pub position: Vec3,
    pub emission: Spd,
}

pub struct Object {
    pub shape: Box<dyn Shape>,
    pub material: Box<dyn Material>,
}

pub struct MovingObject {
    pub shape: Box<dyn Shape>,
    pub material: Box<dyn Material>,
    pub trajectory: Trajectory,
}

pub type SkyFn = Box<dyn Fn(Vec3) -> Spd + Send + Sync>;

pub struct Scene {
    pub name: String,
    pub objects: Vec<Object>,
    pub moving_objects: Vec<MovingObject>,
    pub lights: Vec<Light>,
    pub time: f64,
    pub sky: SkyFn,
}

impl Scene {
    /// Find the closest intersection with any object in the scene.
    pub fn intersect(&self, origin: Vec3, dir: Vec3, t_min: f64, t_max: f64) -> Option<(Hit, &dyn Material)> {
        let mut closest = t_max;
        let mut result: Option<(Hit, &dyn Material)> = None;

        // Static objects
        for obj in &self.objects {
            if let Some(h) = obj.shape.intersect(origin, dir, t_min, closest) {
                closest = h.t;
                result = Some((h, obj.material.as_ref()));
            }
        }

        // Moving objects: solve retarded time, intersect at retarded position
        for mo in &self.moving_objects {
            let Some((t_emit, obj_pos)) = retarded::solve(origin, self.time, &*mo.trajectory) else {
                continue;
            };

            let local_origin = origin - obj_pos;
            if let Some(mut h) = mo.shape.intersect(local_origin, dir, t_min, closest) {
                h.point = h.point + obj_pos;
                h.source_velocity = retarded::velocity(&*mo.trajectory, t_emit);
                closest = h.t;
                result = Some((h, mo.material.as_ref()));
            }
        }

        result
    }
}
