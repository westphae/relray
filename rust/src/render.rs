use std::f64::consts::PI;

use image::{Rgba, RgbaImage};
use rand::rngs::SmallRng;
use rand::{Rng, SeedableRng};
use rayon::prelude::*;

use crate::camera::Camera;
use crate::lorentz;
use crate::scene::Scene;
use crate::spectrum::{self, Spd};
use crate::vec::Vec3;

pub struct Config {
    pub width: u32,
    pub height: u32,
    pub max_depth: u32,
    pub samples_per_px: u32,
}

const TILE_SIZE: u32 = 32;

struct Tile {
    x0: u32,
    y0: u32,
    x1: u32,
    y1: u32,
    seed: u64,
}

/// Render a single frame using tile-based parallel rendering.
pub fn render_frame(cfg: &Config, scene: &Scene, camera: &Camera) -> RgbaImage {
    let mut img = RgbaImage::new(cfg.width, cfg.height);

    // Build tile list
    let mut tiles = Vec::new();
    let mut tile_idx: u64 = 0;
    let mut y = 0;
    while y < cfg.height {
        let mut x = 0;
        while x < cfg.width {
            tiles.push(Tile {
                x0: x,
                y0: y,
                x1: (x + TILE_SIZE).min(cfg.width),
                y1: (y + TILE_SIZE).min(cfg.height),
                seed: tile_idx * 31337,
            });
            tile_idx += 1;
            x += TILE_SIZE;
        }
        y += TILE_SIZE;
    }

    // Render tiles in parallel, collecting pixel data
    let pixel_data: Vec<(u32, u32, [u8; 4])> = tiles
        .par_iter()
        .flat_map(|tile| {
            let mut rng = SmallRng::seed_from_u64(tile.seed);
            let mut pixels = Vec::new();

            let inv_w = 1.0 / cfg.width as f64;
            let inv_h = 1.0 / cfg.height as f64;
            let inv_s = 1.0 / cfg.samples_per_px as f64;

            for y in tile.y0..tile.y1 {
                for x in tile.x0..tile.x1 {
                    let mut acc = Spd::default();
                    for _ in 0..cfg.samples_per_px {
                        let jx: f64 = rng.random();
                        let jy: f64 = rng.random();
                        let u = (x as f64 + jx) * inv_w;
                        let v = 1.0 - (y as f64 + jy) * inv_h;
                        let spd = trace(scene, camera, cfg.max_depth, u, v, &mut rng);
                        acc = acc.add(spd);
                    }
                    acc = acc.scale(inv_s);

                    let (cx, cy, cz) = acc.to_xyz();
                    let (r, g, b) = spectrum::xyz_to_srgb(cx, cy, cz);
                    pixels.push((x, y, [r, g, b, 255]));
                }
            }
            pixels
        })
        .collect();

    // Write pixels to image
    for (x, y, rgba) in pixel_data {
        img.put_pixel(x, y, Rgba(rgba));
    }

    img
}

/// Trace a single pixel at normalized screen coords (u, v).
fn trace(scene: &Scene, camera: &Camera, max_depth: u32, u: f64, v: f64, rng: &mut SmallRng) -> Spd {
    let dir_obs = camera.ray_dir(u, v);
    let ab = lorentz::aberrate(dir_obs, camera.beta);
    let dir_world = ab.dir;
    let doppler = ab.doppler;

    let mut spd = trace_world(scene, camera.position, dir_world, max_depth, rng);

    spd = spd.shift(1.0 / doppler);
    spd = spd.scale(doppler * doppler * doppler);
    spd
}

/// Recursive world-frame ray tracing.
fn trace_world(scene: &Scene, origin: Vec3, dir: Vec3, depth: u32, rng: &mut SmallRng) -> Spd {
    if depth == 0 {
        return Spd::default();
    }

    let Some((hit, mat)) = scene.intersect(origin, dir, 0.001, 1e12) else {
        return (scene.sky)(dir);
    };

    let emitted = mat.emitted(&hit);

    // Direct lighting from point lights
    let mut direct = Spd::default();
    for light in &scene.lights {
        let to_light = light.position - hit.point;
        let dist = to_light.length();
        let light_dir = to_light * (1.0 / dist);

        // Shadow test
        if scene.intersect(hit.point, light_dir, 0.001, dist - 0.001).is_some() {
            continue;
        }

        let cos_theta = hit.normal.dot(light_dir);
        if cos_theta <= 0.0 {
            continue;
        }

        let falloff = cos_theta / (4.0 * PI * dist * dist);
        direct = direct.add(light.emission.scale(falloff));
    }

    let scatter = mat.scatter(dir, &hit, rng);
    let direct_contrib = direct.mul(scatter.reflectance);

    // Indirect lighting (recursive bounce)
    let mut indirect = Spd::default();
    if scatter.scattered && depth > 1 {
        let bounced = trace_world(scene, hit.point, scatter.out_dir, depth - 1, rng);
        indirect = bounced.mul(scatter.reflectance);
    }

    let mut result = emitted.add(direct_contrib).add(indirect);

    // Source Doppler shift for moving objects
    if hit.source_velocity.length_sq() > 0.0 {
        let beta = hit.source_velocity;
        let n_photon = (-dir).normalize();
        let gamma = 1.0 / (1.0 - beta.length_sq()).sqrt();
        let d_source = 1.0 / (gamma * (1.0 - beta.dot(n_photon)));
        result = result.shift(1.0 / d_source);
        result = result.scale(d_source * d_source * d_source);
    }

    result
}
