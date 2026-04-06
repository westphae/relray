mod camera;
mod cie_data;
mod geometry;
mod lorentz;
mod material;
mod output;
mod render;
mod retarded;
mod scene;
mod scenefile;
mod spectrum;
mod vars;
mod vec;

use std::f64::consts::PI;
use std::time::Instant;

use clap::{Parser, Subcommand};

use crate::camera::Camera;
use crate::geometry::{BoxShape, Plane, Shape, Sphere, Transformed};
use crate::material::{Checker, CheckerSphere, Diffuse, Glass, Material, Mirror};
use crate::output::{assemble_video, save_png};
use crate::render::{render_frame, Config};
use crate::scene::{Light, MovingObject, Object, Scene, SkyFn};
use crate::spectrum::Spd;
use crate::vec::{Mat3, Vec3};

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

#[derive(Parser)]
#[command(name = "rrelray", about = "Relativistic ray tracer")]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Render a single static image
    Render {
        /// Set variable: name=value (repeatable)
        #[arg(long = "var")]
        vars: Vec<String>,
        #[command(flatten)]
        common: CommonArgs,
    },
    /// Render a video sweeping variables across a range
    Sweep {
        /// Sweep variable: name:start:end (repeatable)
        #[arg(long = "range")]
        ranges: Vec<String>,
        /// Number of frames
        #[arg(long, default_value_t = 200)]
        steps: usize,
        #[arg(long, default_value_t = 30)]
        fps: u32,
        #[command(flatten)]
        common: CommonArgs,
    },
    /// Render a walk-through video
    Walk {
        #[arg(long, default_value_t = 10.0)]
        duration: f64,
        #[arg(long, default_value_t = 0.5)]
        speed: f64,
        #[arg(long, default_value_t = 30)]
        fps: u32,
        #[command(flatten)]
        common: CommonArgs,
    },
}

#[derive(clap::Args)]
struct CommonArgs {
    #[arg(long, default_value_t = 800)]
    width: u32,
    #[arg(long, default_value_t = 600)]
    height: u32,
    #[arg(long, default_value_t = 32)]
    samples: u32,
    #[arg(long, default_value_t = 8)]
    depth: u32,
    #[arg(long, default_value = "spheres")]
    scene: String,
    #[arg(long)]
    file: Option<String>,
    #[arg(long)]
    out: Option<String>,
}

impl CommonArgs {
    fn config(&self) -> Config {
        Config {
            width: self.width,
            height: self.height,
            max_depth: self.depth,
            samples_per_px: self.samples,
        }
    }

    fn load_scene_with_vars(&self, vars: &std::collections::HashMap<String, f64>) -> (Scene, Option<Camera>) {
        if let Some(ref path) = self.file {
            match scenefile::load_with_vars(path, vars) {
                Ok((sc, cam)) => return (sc, cam),
                Err(e) => {
                    eprintln!("Error loading scene file: {}", e);
                    std::process::exit(1);
                }
            }
        }
        match self.scene.as_str() {
            "room" => (build_room_scene(), None),
            _ => (build_spheres_scene(), None),
        }
    }
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

fn main() {
    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Render { vars: var_flags, common }) => {
            let vars = parse_var_flags(&var_flags);
            let (sc, file_cam) = common.load_scene_with_vars(&vars);
            let out = common.out.clone().unwrap_or_else(|| "output.png".into());
            let cam = file_cam.unwrap_or_else(|| camera_preset(&sc.name, common.width, common.height));
            let aspect = common.width as f64 / common.height as f64;
            let cam = Camera::new(cam.position, cam.look_at, cam.up, cam.vfov, aspect, cam.velocity);
            run_single(&common.config(), &sc, &cam, &out);
        }
        Some(Commands::Sweep {
            ranges: range_flags,
            steps,
            fps,
            common,
        }) => {
            let file = common.file.clone().expect("sweep requires --file with a YAML scene containing $variables");
            let ranges = parse_range_flags(&range_flags);
            let out = common.out.clone().unwrap_or_else(|| "sweep.mp4".into());
            run_sweep(&common.config(), &file, common.width, common.height, &ranges, steps, fps, &out);
        }
        Some(Commands::Walk {
            duration,
            speed,
            fps,
            common,
        }) => {
            let (mut sc, _) = common.load_scene_with_vars(&std::collections::HashMap::new());
            let out = common.out.clone().unwrap_or_else(|| "walk.mp4".into());
            run_walk(&common.config(), &mut sc, common.width, common.height, duration, speed, fps, &out);
        }
        None => {
            eprintln!(
                "Usage: rrelray <command> [flags]\n\n\
                 Relativistic ray tracer -- renders scenes with physically correct\n\
                 aberration, Doppler shift, and searchlight effects.\n\n\
                 Commands:\n  \
                   render    Render a single static image (default)\n  \
                   sweep     Render a video sweeping variables across a range\n  \
                   walk      Render a first-person walk-through video\n\n\
                 Run 'rrelray <command> --help' for command-specific flags."
            );
        }
    }
}

// ---------------------------------------------------------------------------
// Camera preset
// ---------------------------------------------------------------------------

fn parse_var_flags(flags: &[String]) -> std::collections::HashMap<String, f64> {
    let mut map = std::collections::HashMap::new();
    for f in flags {
        match vars::parse_var(f) {
            Ok((name, val)) => { map.insert(name, val); }
            Err(e) => { eprintln!("{}", e); std::process::exit(1); }
        }
    }
    map
}

fn parse_range_flags(flags: &[String]) -> Vec<vars::VarRange> {
    flags.iter().map(|f| {
        vars::parse_range(f).unwrap_or_else(|e| { eprintln!("{}", e); std::process::exit(1); })
    }).collect()
}

fn camera_preset(scene_name: &str, width: u32, height: u32) -> Camera {
    let aspect = width as f64 / height as f64;
    match scene_name {
        "room" => Camera::new(
            Vec3::new(0.0, 1.0, -0.5),
            Vec3::new(0.0, 0.8, 3.0),
            Vec3::new(0.0, 1.0, 0.0),
            70.0,
            aspect,
            Vec3::default(),
        ),
        _ => Camera::new(
            Vec3::new(0.0, 0.5, -3.0),
            Vec3::new(0.0, 0.3, 0.0),
            Vec3::new(0.0, 1.0, 0.0),
            60.0,
            aspect,
            Vec3::default(),
        ),
    }
}

// ---------------------------------------------------------------------------
// Run modes
// ---------------------------------------------------------------------------

fn run_single(cfg: &Config, sc: &Scene, cam: &Camera, out_file: &str) {
    let v = cam.velocity;
    println!(
        "Rendering {}x{}, velocity=[{:.3},{:.3},{:.3}], {} spp, {} bounces",
        cfg.width, cfg.height, v.x, v.y, v.z, cfg.samples_per_px, cfg.max_depth
    );

    let start = Instant::now();
    let img = render_frame(cfg, sc, cam);
    println!("Rendered in {:?}", start.elapsed());

    if let Err(e) = save_png(out_file, &img) {
        eprintln!("Error saving PNG: {}", e);
        std::process::exit(1);
    }
    println!("Saved to {}", out_file);
}

fn run_sweep(
    cfg: &Config,
    file: &str,
    width: u32,
    height: u32,
    ranges: &[vars::VarRange],
    steps: usize,
    fps: u32,
    out_file: &str,
) {
    for r in ranges {
        println!("  {}: {:.4} → {:.4}", r.name, r.start, r.end);
    }
    println!(
        "Sweep: {} steps, {}x{}, {} spp, {} bounces, {} fps",
        steps, cfg.width, cfg.height, cfg.samples_per_px, cfg.max_depth, fps
    );

    let frame_dir = tempfile::tempdir().expect("failed to create temp dir");
    let aspect = width as f64 / height as f64;
    let total_start = Instant::now();

    for i in 0..steps {
        let t = if steps > 1 { i as f64 / (steps - 1) as f64 } else { 0.0 };
        let var_vals = vars::interpolate_vars(ranges, t);

        let (sc, file_cam) = match scenefile::load_with_vars(file, &var_vals) {
            Ok(r) => r,
            Err(e) => { eprintln!("frame {}: {}", i, e); std::process::exit(1); }
        };
        let cam = file_cam.expect("sweep requires a camera defined in the YAML scene file");
        let cam = Camera::new(cam.position, cam.look_at, cam.up, cam.vfov, aspect, cam.velocity);

        let start = Instant::now();
        let img = render_frame(cfg, &sc, &cam);
        let elapsed = start.elapsed();

        let frame_path = frame_dir.path().join(format!("frame_{:04}.png", i));
        if let Err(e) = save_png(frame_path.to_str().unwrap(), &img) {
            eprintln!("Error saving frame: {}", e);
            std::process::exit(1);
        }

        let var_str: String = ranges.iter()
            .map(|r| format!("  {}={:+.4}", r.name, var_vals[&r.name]))
            .collect::<Vec<_>>()
            .join("");
        println!("Frame {}/{}{}  {:?}", i + 1, steps, var_str, elapsed);
    }

    println!("All frames rendered in {:?}", total_start.elapsed());
    println!("Assembling video...");

    let pattern = frame_dir
        .path()
        .join("frame_%04d.png")
        .to_str()
        .unwrap()
        .to_string();
    if let Err(e) = assemble_video(&pattern, fps, out_file) {
        eprintln!("ffmpeg failed: {}", e);
        std::process::exit(1);
    }
    println!("Saved to {}", out_file);
}

fn run_walk(
    cfg: &Config,
    sc: &mut Scene,
    width: u32,
    height: u32,
    duration: f64,
    speed: f64,
    fps: u32,
    out_file: &str,
) {
    let num_frames = (duration * fps as f64) as usize;
    let dt = 1.0 / fps as f64;
    println!(
        "Walk-through: {:.1}s at speed {:.2} c, {} frames",
        duration, speed, num_frames
    );
    println!(
        "Rendering {}x{}, {} spp, {} bounces, {} fps",
        cfg.width, cfg.height, cfg.samples_per_px, cfg.max_depth, fps
    );

    let frame_dir = tempfile::tempdir().expect("failed to create temp dir");
    let start_z = -2.0;
    let eye_y = 1.0;
    let aspect = width as f64 / height as f64;

    let total_start = Instant::now();

    for i in 0..num_frames {
        let t = i as f64 * dt;
        let z = start_z + speed * t;

        sc.time = t;

        let cam = Camera::new(
            Vec3::new(0.0, eye_y, z),
            Vec3::new(0.0, eye_y - 0.1, z + 2.0),
            Vec3::new(0.0, 1.0, 0.0),
            70.0,
            aspect,
            Vec3::new(0.0, 0.0, speed),
        );

        let start = Instant::now();
        let img = render_frame(cfg, sc, &cam);
        let elapsed = start.elapsed();

        let frame_path = frame_dir.path().join(format!("frame_{:05}.png", i));
        if let Err(e) = save_png(frame_path.to_str().unwrap(), &img) {
            eprintln!("Error saving frame: {}", e);
            std::process::exit(1);
        }

        println!(
            "Frame {}/{} t={:.2}s z={:.2} {:?}",
            i + 1,
            num_frames,
            t,
            z,
            elapsed
        );
    }

    println!("All frames rendered in {:?}", total_start.elapsed());
    println!("Assembling video...");

    let pattern = frame_dir
        .path()
        .join("frame_%05d.png")
        .to_str()
        .unwrap()
        .to_string();
    if let Err(e) = assemble_video(&pattern, fps, out_file) {
        eprintln!("ffmpeg failed: {}", e);
        std::process::exit(1);
    }
    println!("Saved to {}", out_file);
}

// ---------------------------------------------------------------------------
// Shape helpers
// ---------------------------------------------------------------------------

/// Position a shape at (x, y, z) with no rotation.
fn at(shape: Box<dyn Shape>, x: f64, y: f64, z: f64) -> Box<dyn Shape> {
    Box::new(Transformed::new(
        shape,
        Vec3::new(x, y, z),
        Mat3::identity(),
    ))
}

/// Position and rotate a shape (Euler angles in degrees: yaw, pitch, roll).
fn at_rot(
    shape: Box<dyn Shape>,
    x: f64,
    y: f64,
    z: f64,
    yaw: f64,
    pitch: f64,
    roll: f64,
) -> Box<dyn Shape> {
    Box::new(Transformed::new(
        shape,
        Vec3::new(x, y, z),
        Mat3::from_euler_deg(yaw, pitch, roll),
    ))
}

/// Create a box shape positioned at the given center.
fn bx(w: f64, h: f64, d: f64, cx: f64, cy: f64, cz: f64) -> Box<dyn Shape> {
    at(
        Box::new(BoxShape {
            size: Vec3::new(w, h, d),
        }),
        cx,
        cy,
        cz,
    )
}

// ---------------------------------------------------------------------------
// Built-in scenes
// ---------------------------------------------------------------------------

fn build_spheres_scene() -> Scene {
    let sunlight = spectrum::blackbody(5778.0, 1.0);
    let fill_light = spectrum::blackbody(7500.0, 1.0);
    let sky_base = spectrum::blackbody(12000.0, 1.0);

    let red: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.8, 0.1, 0.1),
    });
    let green: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.1, 0.8, 0.1),
    });
    let blue: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.1, 0.1, 0.8),
    });
    let mirror: Box<dyn Material> = Box::new(Mirror {
        reflectance: spectrum::constant(0.95),
    });
    let glass: Box<dyn Material> = Box::new(Glass {
        ior: 1.5,
        tint: spectrum::constant(1.0),
    });
    let floor: Box<dyn Material> = Box::new(Checker {
        even: spectrum::from_rgb(0.7, 0.7, 0.7),
        odd: spectrum::from_rgb(0.15, 0.15, 0.15),
        scale: 0.5,
    });

    let sky_fn: SkyFn = Box::new(move |dir: Vec3| {
        let mut t = 0.5 * (dir.y + 1.0);
        if t < 0.0 {
            t = 0.0;
        }
        sky_base.scale(0.15 * t)
    });

    Scene {
        name: "spheres".into(),
        objects: vec![
            Object {
                shape: at(Box::new(Plane), 0.0, -0.5, 0.0),
                material: floor,
            },
            Object {
                shape: at(Box::new(Sphere { radius: 0.5 }), -1.8, 0.0, 1.5),
                material: red,
            },
            Object {
                shape: at(Box::new(Sphere { radius: 0.5 }), -0.6, 0.0, 2.0),
                material: green,
            },
            Object {
                shape: at(Box::new(Sphere { radius: 0.5 }), 0.6, 0.0, 2.0),
                material: mirror,
            },
            Object {
                shape: at(Box::new(Sphere { radius: 0.5 }), 1.8, 0.0, 1.5),
                material: glass,
            },
            Object {
                shape: at(Box::new(Sphere { radius: 0.2 }), 0.0, -0.3, 1.0),
                material: blue,
            },
        ],
        moving_objects: vec![],
        lights: vec![
            Light {
                position: Vec3::new(2.0, 5.0, 0.0),
                emission: sunlight.scale(15.0),
            },
            Light {
                position: Vec3::new(-3.0, 3.0, -2.0),
                emission: fill_light.scale(8.0),
            },
        ],
        time: 0.0,
        sky: sky_fn,
    }
}

fn build_room_scene() -> Scene {
    let sunlight = spectrum::blackbody(5778.0, 1.0);
    let warm_light = spectrum::blackbody(3500.0, 1.0);

    let wall_white: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.85, 0.82, 0.78),
    });
    let wall_white2: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.85, 0.82, 0.78),
    });
    let wall_white3: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.85, 0.82, 0.78),
    });
    let wall_accent: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.6, 0.15, 0.1),
    });
    let glass_mat: Box<dyn Material> = Box::new(Glass {
        ior: 1.5,
        tint: spectrum::constant(1.0),
    });
    let mirror_mat: Box<dyn Material> = Box::new(Mirror {
        reflectance: spectrum::constant(0.92),
    });
    let floor_wood: Box<dyn Material> = Box::new(Checker {
        even: spectrum::from_rgb(0.55, 0.35, 0.18),
        odd: spectrum::from_rgb(0.65, 0.45, 0.25),
        scale: 0.4,
    });
    let ceiling: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.9, 0.9, 0.9),
    });
    let furniture: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.3, 0.2, 0.1),
    });
    let furniture2: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.3, 0.2, 0.1),
    });
    let furniture3: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.3, 0.2, 0.1),
    });
    let cushion: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.15, 0.25, 0.5),
    });
    let table_mat: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.4, 0.25, 0.12),
    });
    let ball_mat: Box<dyn Material> = Box::new(Diffuse {
        reflectance: spectrum::from_rgb(0.9, 0.2, 0.2),
    });

    let sky_fn: SkyFn = Box::new(|_: Vec3| Spd::default());

    Scene {
        name: "room".into(),
        objects: vec![
            // Floor (Y=0)
            Object {
                shape: at(Box::new(Plane), 0.0, 0.0, 0.0),
                material: floor_wood,
            },
            // Ceiling (Y=2.5, flip)
            Object {
                shape: at_rot(Box::new(Plane), 0.0, 2.5, 0.0, 0.0, 0.0, 180.0),
                material: ceiling,
            },
            // Back wall (Z=6)
            Object {
                shape: at_rot(Box::new(Plane), 0.0, 0.0, 6.0, 0.0, -90.0, 0.0),
                material: wall_accent,
            },
            // Left wall (X=-3)
            Object {
                shape: at_rot(Box::new(Plane), -3.0, 0.0, 0.0, 0.0, 0.0, 90.0),
                material: wall_white,
            },
            // Right wall (X=3)
            Object {
                shape: at_rot(Box::new(Plane), 3.0, 0.0, 0.0, 0.0, 0.0, -90.0),
                material: wall_white2,
            },
            // Front wall (Z=-2)
            Object {
                shape: at_rot(Box::new(Plane), 0.0, 0.0, -2.0, 0.0, 90.0, 0.0),
                material: wall_white3,
            },
            // Coffee table
            Object {
                shape: bx(1.0, 0.4, 1.0, 0.0, 0.2, 3.0),
                material: table_mat,
            },
            // Couch base
            Object {
                shape: bx(1.3, 0.45, 3.0, -2.15, 0.225, 3.0),
                material: furniture,
            },
            // Couch back
            Object {
                shape: bx(0.3, 0.45, 3.0, -2.65, 0.675, 3.0),
                material: furniture2,
            },
            // Couch cushion
            Object {
                shape: bx(0.9, 0.1, 2.6, -2.05, 0.5, 3.0),
                material: cushion,
            },
            // Bookshelf
            Object {
                shape: bx(1.0, 1.8, 1.8, 2.3, 0.9, 4.9),
                material: furniture3,
            },
            // Glass globe on coffee table
            Object {
                shape: at(Box::new(Sphere { radius: 0.12 }), 0.1, 0.55, 3.0),
                material: glass_mat,
            },
            // Mirror sphere on coffee table
            Object {
                shape: at(Box::new(Sphere { radius: 0.08 }), -0.2, 0.52, 2.8),
                material: mirror_mat,
            },
            // Red decorative ball
            Object {
                shape: at(Box::new(Sphere { radius: 0.08 }), 0.3, 0.5, 3.2),
                material: ball_mat,
            },
        ],
        moving_objects: vec![MovingObject {
            shape: Box::new(Sphere { radius: 0.12 }),
            material: Box::new(CheckerSphere {
                even: spectrum::from_rgb(0.9, 0.85, 0.15),
                odd: spectrum::from_rgb(0.1, 0.1, 0.6),
                num_squares: 8,
            }),
            trajectory: {
                let orbit_radius = 0.4;
                let orbit_period = 10.0;
                let center_x = 0.0;
                let center_z = 3.0;
                let height = 1.2;
                Box::new(move |t: f64| {
                    let angle = 2.0 * PI * t / orbit_period;
                    Vec3::new(
                        center_x + orbit_radius * angle.cos(),
                        height,
                        center_z + orbit_radius * angle.sin(),
                    )
                })
            },
        }],
        lights: vec![
            Light {
                position: Vec3::new(2.5, 2.0, 3.0),
                emission: sunlight.scale(25.0),
            },
            Light {
                position: Vec3::new(0.0, 2.3, 3.0),
                emission: warm_light.scale(12.0),
            },
            Light {
                position: Vec3::new(2.3, 1.9, 4.9),
                emission: warm_light.scale(5.0),
            },
        ],
        time: 0.0,
        sky: sky_fn,
    }
}
