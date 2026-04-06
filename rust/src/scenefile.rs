use std::f64::consts::PI;

use serde::Deserialize;

use crate::camera::Camera;
use crate::geometry::{
    BoxShape, Cone, Cylinder, Disk, Plane, Pyramid, Shape, Sphere, Transformed, Triangle,
};
use crate::material::{Checker, CheckerSphere, Diffuse, Glass, Material, Mirror};
use crate::retarded::{Trajectory, C};
use crate::scene::{Light, MovingObject, Object, Scene, SkyFn};
use crate::spectrum::{self, Spd};
use crate::vec::{Mat3, Vec3};

// ---------------------------------------------------------------------------
// YAML intermediate types
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
pub struct SceneFile {
    pub name: Option<String>,
    pub camera: Option<CameraDef>,
    pub sky: Option<SkyDef>,
    #[serde(default)]
    pub lights: Vec<LightDef>,
    #[serde(default)]
    pub objects: Vec<ObjectDef>,
    #[serde(default)]
    pub moving_objects: Vec<MovingObjectDef>,
}

#[derive(Deserialize)]
pub struct CameraDef {
    pub position: [f64; 3],
    pub look_at: [f64; 3],
    pub up: [f64; 3],
    pub vfov: f64,
}

#[derive(Deserialize)]
pub struct LightDef {
    pub position: [f64; 3],
    pub emission: SpdDef,
}

#[derive(Deserialize)]
pub struct ObjectDef {
    pub shape: ShapeDef,
    pub material: MaterialDef,
}

#[derive(Deserialize)]
pub struct MovingObjectDef {
    pub shape: ShapeDef,
    pub material: MaterialDef,
    pub trajectory: TrajectoryDef,
}

#[derive(Deserialize)]
pub struct ShapeDef {
    pub sphere: Option<SphereDef>,
    pub plane: Option<PlaneDef>,
    #[serde(rename = "box")]
    pub box_shape: Option<BoxDef>,
    pub cylinder: Option<CylinderDef>,
    pub cone: Option<ConeDef>,
    pub disk: Option<DiskDef>,
    pub triangle: Option<TriangleDef>,
    pub pyramid: Option<PyramidDef>,
    pub position: Option<[f64; 3]>,
    pub rotation: Option<[f64; 3]>,
}

#[derive(Deserialize)]
pub struct PlaneDef {}

#[derive(Deserialize)]
pub struct SphereDef {
    pub radius: f64,
}

#[derive(Deserialize)]
pub struct BoxDef {
    pub size: [f64; 3],
}

#[derive(Deserialize)]
pub struct CylinderDef {
    pub radius: f64,
    pub height: f64,
}

#[derive(Deserialize)]
pub struct ConeDef {
    pub radius: f64,
    pub height: f64,
}

#[derive(Deserialize)]
pub struct DiskDef {
    pub radius: f64,
}

#[derive(Deserialize)]
pub struct TriangleDef {
    pub v0: [f64; 3],
    pub v1: [f64; 3],
    pub v2: [f64; 3],
}

#[derive(Deserialize)]
pub struct PyramidDef {
    pub base_radius: f64,
    pub height: f64,
    pub sides: usize,
}

// Material: tagged union — exactly one field set.
// Note: diffuse and mirror in the YAML use inline SPD (e.g. `diffuse: { rgb: [...] }`),
// so they deserialize directly as SpdDef.
#[derive(Deserialize)]
pub struct MaterialDef {
    pub diffuse: Option<SpdDef>,
    pub mirror: Option<SpdDef>,
    pub glass: Option<GlassDef>,
    pub checker: Option<CheckerDef>,
    pub checker_sphere: Option<CheckerSphereDef>,
}

#[derive(Deserialize)]
pub struct GlassDef {
    pub ior: f64,
    pub tint: SpdDef,
}

#[derive(Deserialize)]
pub struct CheckerDef {
    pub even: SpdDef,
    pub odd: SpdDef,
    pub scale: f64,
}

#[derive(Deserialize)]
pub struct CheckerSphereDef {
    pub even: SpdDef,
    pub odd: SpdDef,
    pub num_squares: i32,
}

// SPD: tagged union — exactly one field set.
#[derive(Deserialize)]
pub struct SpdDef {
    pub rgb: Option<[f64; 3]>,
    pub blackbody: Option<BlackbodyDef>,
    pub constant: Option<f64>,
    pub d65: Option<f64>,
    pub monochromatic: Option<MonochromaticDef>,
    pub reflectance: Option<Vec<[f64; 2]>>,
}

#[derive(Deserialize)]
pub struct BlackbodyDef {
    pub temp: f64,
    pub luminance: f64,
}

#[derive(Deserialize)]
pub struct MonochromaticDef {
    pub wavelength: f64,
    pub power: f64,
}

// Trajectory: tagged union.
#[derive(Deserialize)]
pub struct TrajectoryDef {
    #[serde(rename = "static")]
    pub static_traj: Option<StaticDef>,
    pub linear: Option<LinearDef>,
    pub orbit: Option<OrbitDef>,
}

#[derive(Deserialize)]
pub struct StaticDef {
    pub position: [f64; 3],
}

#[derive(Deserialize)]
pub struct LinearDef {
    pub start: [f64; 3],
    pub velocity: [f64; 3],
}

#[derive(Deserialize)]
pub struct OrbitDef {
    pub center: [f64; 3],
    pub radius: f64,
    pub period: f64,
    pub axis: Option<String>,
}

// Sky definition.
#[derive(Deserialize)]
pub struct SkyDef {
    #[serde(rename = "type")]
    pub sky_type: Option<String>,
    pub top: Option<SpdDef>,
    pub bottom: Option<SpdDef>,
    pub emission: Option<SpdDef>,
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

pub fn load(path: &str) -> Result<(Scene, Option<Camera>), Box<dyn std::error::Error>> {
    let data = std::fs::read_to_string(path)?;
    let sf: SceneFile = serde_yaml::from_str(&data)?;
    convert(&sf)
}

// ---------------------------------------------------------------------------
// Conversion from YAML types to runtime types
// ---------------------------------------------------------------------------

fn convert(sf: &SceneFile) -> Result<(Scene, Option<Camera>), Box<dyn std::error::Error>> {
    let name = sf.name.clone().unwrap_or_default();

    // Lights
    let mut lights = Vec::with_capacity(sf.lights.len());
    for (i, ls) in sf.lights.iter().enumerate() {
        let spd = convert_spd(&ls.emission)
            .map_err(|e| format!("lights[{}].emission: {}", i, e))?;
        lights.push(Light {
            position: v3(ls.position),
            emission: spd,
        });
    }

    // Static objects
    let mut objects = Vec::with_capacity(sf.objects.len());
    for (i, os) in sf.objects.iter().enumerate() {
        let shape = convert_shape(&os.shape)
            .map_err(|e| format!("objects[{}].shape: {}", i, e))?;
        let mat = convert_material(&os.material)
            .map_err(|e| format!("objects[{}].material: {}", i, e))?;
        objects.push(Object {
            shape,
            material: mat,
        });
    }

    // Moving objects
    let mut moving_objects = Vec::with_capacity(sf.moving_objects.len());
    for (i, ms) in sf.moving_objects.iter().enumerate() {
        let shape = convert_shape(&ms.shape)
            .map_err(|e| format!("moving_objects[{}].shape: {}", i, e))?;
        let mat = convert_material(&ms.material)
            .map_err(|e| format!("moving_objects[{}].material: {}", i, e))?;
        let traj = convert_trajectory(&ms.trajectory)
            .map_err(|e| format!("moving_objects[{}].trajectory: {}", i, e))?;
        moving_objects.push(MovingObject {
            shape,
            material: mat,
            trajectory: traj,
        });
    }

    // Sky
    let sky: SkyFn = if let Some(ref s) = sf.sky {
        convert_sky(s).map_err(|e| format!("sky: {}", e))?
    } else {
        Box::new(|_: Vec3| Spd::default())
    };

    // Camera
    let cam = sf.camera.as_ref().map(|c| {
        Camera::new(
            v3(c.position),
            v3(c.look_at),
            v3(c.up),
            c.vfov,
            1.0, // aspect will be overridden by caller
            Vec3::default(),
        )
    });

    let scene = Scene {
        name,
        objects,
        moving_objects,
        lights,
        time: 0.0,
        sky,
    };

    Ok((scene, cam))
}

// ---------------------------------------------------------------------------
// SPD
// ---------------------------------------------------------------------------

fn convert_spd(s: &SpdDef) -> Result<Spd, String> {
    if let Some(rgb) = &s.rgb {
        return Ok(spectrum::from_rgb(rgb[0], rgb[1], rgb[2]));
    }
    if let Some(bb) = &s.blackbody {
        return Ok(spectrum::blackbody(bb.temp, bb.luminance));
    }
    if let Some(c) = s.constant {
        return Ok(spectrum::constant(c));
    }
    if let Some(d) = s.d65 {
        return Ok(spectrum::d65().scale(d));
    }
    if let Some(m) = &s.monochromatic {
        return Ok(spectrum::monochromatic(m.wavelength, m.power));
    }
    if let Some(r) = &s.reflectance {
        return Ok(spectrum::from_reflectance_curve(r));
    }
    Err("no SPD type specified (use rgb, blackbody, constant, d65, monochromatic, or reflectance)".into())
}

// ---------------------------------------------------------------------------
// Shapes
// ---------------------------------------------------------------------------

fn convert_shape(s: &ShapeDef) -> Result<Box<dyn Shape>, String> {
    let shape: Box<dyn Shape> = if let Some(sp) = &s.sphere {
        Box::new(Sphere { radius: sp.radius })
    } else if s.plane.is_some() {
        Box::new(Plane)
    } else if let Some(b) = &s.box_shape {
        Box::new(BoxShape {
            size: v3(b.size),
        })
    } else if let Some(cy) = &s.cylinder {
        Box::new(Cylinder {
            radius: cy.radius,
            height: cy.height,
        })
    } else if let Some(co) = &s.cone {
        Box::new(Cone {
            radius: co.radius,
            height: co.height,
        })
    } else if let Some(d) = &s.disk {
        Box::new(Disk { radius: d.radius })
    } else if let Some(tr) = &s.triangle {
        Box::new(Triangle {
            v0: v3(tr.v0),
            v1: v3(tr.v1),
            v2: v3(tr.v2),
        })
    } else if let Some(py) = &s.pyramid {
        Box::new(Pyramid::new(py.base_radius, py.height, py.sides))
    } else {
        return Err(
            "no shape type specified (use sphere, plane, box, cylinder, cone, disk, triangle, or pyramid)".into(),
        );
    };

    // Apply optional transform
    if s.position.is_some() || s.rotation.is_some() {
        let pos = s.position.map_or(Vec3::default(), v3);
        let rot = s.rotation.map_or(Mat3::identity(), |r| {
            Mat3::from_euler_deg(r[0], r[1], r[2])
        });
        Ok(Box::new(Transformed::new(shape, pos, rot)))
    } else {
        Ok(shape)
    }
}

// ---------------------------------------------------------------------------
// Materials
// ---------------------------------------------------------------------------

fn convert_material(m: &MaterialDef) -> Result<Box<dyn Material>, String> {
    if let Some(ref d) = m.diffuse {
        let spd = convert_spd(d).map_err(|e| format!("diffuse: {}", e))?;
        return Ok(Box::new(Diffuse { reflectance: spd }));
    }
    if let Some(ref mr) = m.mirror {
        let spd = convert_spd(mr).map_err(|e| format!("mirror: {}", e))?;
        return Ok(Box::new(Mirror { reflectance: spd }));
    }
    if let Some(ref g) = m.glass {
        let tint = convert_spd(&g.tint).map_err(|e| format!("glass.tint: {}", e))?;
        return Ok(Box::new(Glass {
            ior: g.ior,
            tint,
        }));
    }
    if let Some(ref ch) = m.checker {
        let even = convert_spd(&ch.even).map_err(|e| format!("checker.even: {}", e))?;
        let odd = convert_spd(&ch.odd).map_err(|e| format!("checker.odd: {}", e))?;
        return Ok(Box::new(Checker {
            even,
            odd,
            scale: ch.scale,
        }));
    }
    if let Some(ref cs) = m.checker_sphere {
        let even = convert_spd(&cs.even).map_err(|e| format!("checker_sphere.even: {}", e))?;
        let odd = convert_spd(&cs.odd).map_err(|e| format!("checker_sphere.odd: {}", e))?;
        return Ok(Box::new(CheckerSphere {
            even,
            odd,
            num_squares: cs.num_squares,
        }));
    }
    Err("no material type specified (use diffuse, mirror, glass, checker, or checker_sphere)".into())
}

// ---------------------------------------------------------------------------
// Trajectories
// ---------------------------------------------------------------------------

fn convert_trajectory(t: &TrajectoryDef) -> Result<Trajectory, String> {
    if let Some(ref st) = t.static_traj {
        let pos = v3(st.position);
        return Ok(Box::new(move |_: f64| pos));
    }
    if let Some(ref li) = t.linear {
        let start = v3(li.start);
        let vel = v3(li.velocity);
        let speed = vel.length();
        if speed >= C {
            return Err(format!("linear: speed {:.3} exceeds c ({:.1})", speed, C));
        }
        return Ok(Box::new(move |t: f64| start + vel * t));
    }
    if let Some(ref o) = t.orbit {
        let max_speed = 2.0 * PI * o.radius / o.period;
        if max_speed >= C {
            return Err(format!(
                "orbit: max speed {:.3} exceeds c ({:.1})",
                max_speed, C
            ));
        }
        let center = v3(o.center);
        let radius = o.radius;
        let period = o.period;
        let axis = o.axis.clone().unwrap_or_else(|| "y".into());
        return Ok(make_orbit_trajectory(center, radius, period, &axis));
    }
    Err("no trajectory type specified (use static, linear, or orbit)".into())
}

fn make_orbit_trajectory(center: Vec3, radius: f64, period: f64, axis: &str) -> Trajectory {
    let axis = axis.to_owned();
    Box::new(move |t: f64| {
        let angle = 2.0 * PI * t / period;
        let (cos, sin) = (angle.cos(), angle.sin());
        match axis.as_str() {
            "x" => Vec3::new(center.x, center.y + radius * cos, center.z + radius * sin),
            "z" => Vec3::new(center.x + radius * cos, center.y + radius * sin, center.z),
            _ => Vec3::new(center.x + radius * cos, center.y, center.z + radius * sin), // "y" default
        }
    })
}

// ---------------------------------------------------------------------------
// Sky
// ---------------------------------------------------------------------------

fn convert_sky(s: &SkyDef) -> Result<SkyFn, String> {
    let sky_type = s.sky_type.as_deref().unwrap_or("none");
    match sky_type {
        "none" | "" => Ok(Box::new(|_: Vec3| Spd::default())),

        "uniform" => {
            let spd_def = s
                .emission
                .as_ref()
                .ok_or("uniform sky requires 'emission'")?;
            let spd = convert_spd(spd_def)?;
            Ok(Box::new(move |_: Vec3| spd.clone()))
        }

        "gradient" => {
            let top_def = s.top.as_ref().ok_or("gradient sky requires 'top'")?;
            let bot_def = s.bottom.as_ref().ok_or("gradient sky requires 'bottom'")?;
            let top = convert_spd(top_def)?;
            let bot = convert_spd(bot_def)?;
            Ok(Box::new(move |dir: Vec3| {
                let mut t = 0.5 * (dir.y + 1.0);
                if t < 0.0 {
                    t = 0.0;
                }
                if t > 1.0 {
                    t = 1.0;
                }
                bot.scale(1.0 - t).add(top.scale(t))
            }))
        }

        other => Err(format!(
            "unknown sky type {:?} (use none, uniform, or gradient)",
            other
        )),
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn v3(a: [f64; 3]) -> Vec3 {
    Vec3::new(a[0], a[1], a[2])
}
