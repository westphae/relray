use crate::vec::Vec3;

/// Speed of light in scene units per second (1 m/s in our slow-light universe).
pub const C: f64 = 1.0;

/// A trajectory describes the position of an object as a function of time.
pub type Trajectory = Box<dyn Fn(f64) -> Vec3 + Send + Sync>;

/// Solve finds the emission time for a photon arriving at obs_pos at t_obs,
/// emitted by an object following the given trajectory.
///
/// Uses Newton's method on f(t) = |obs_pos - traj(t)|² - C²*(t_obs - t)².
/// Returns (t_emit, obj_pos) or None if the solver fails to converge.
pub fn solve(obs_pos: Vec3, t_obs: f64, traj: &dyn Fn(f64) -> Vec3) -> Option<(f64, Vec3)> {
    let pos0 = traj(t_obs);
    let dist0 = (obs_pos - pos0).length();
    let mut t_emit = t_obs - dist0 / C;

    const MAX_ITER: usize = 50;
    const TOL: f64 = 1e-10;
    const DT: f64 = 1e-8;

    for _ in 0..MAX_ITER {
        let obj_pos = traj(t_emit);
        let delta = obs_pos - obj_pos;
        let dist = delta.length();
        let time_diff = t_obs - t_emit;

        let f = dist * dist - C * C * time_diff * time_diff;

        if f.abs() < TOL {
            return Some((t_emit, obj_pos));
        }

        let pos_plus = traj(t_emit + DT);
        let vel = (pos_plus - obj_pos) * (1.0 / DT);
        let fp = -2.0 * delta.dot(vel) + 2.0 * C * C * time_diff;

        if fp.abs() < 1e-20 {
            break;
        }

        t_emit -= f / fp;

        if t_emit > t_obs {
            t_emit = t_obs - 1e-6;
        }
    }

    // Check convergence
    let obj_pos = traj(t_emit);
    let dist = (obs_pos - obj_pos).length();
    let time_diff = t_obs - t_emit;
    let residual = (dist - C * time_diff).abs();
    if residual < 1e-6 {
        Some((t_emit, obj_pos))
    } else {
        None
    }
}

/// Compute the velocity of a trajectory at time t via numerical differentiation.
pub fn velocity(traj: &dyn Fn(f64) -> Vec3, t: f64) -> Vec3 {
    const DT: f64 = 1e-8;
    let p0 = traj(t);
    let p1 = traj(t + DT);
    (p1 - p0) * (1.0 / DT)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_solve_stationary() {
        let traj = |_t: f64| Vec3::new(0.0, 0.0, 5.0);
        let result = solve(Vec3::default(), 10.0, &traj);
        let (t_emit, pos) = result.unwrap();
        assert!((t_emit - 5.0).abs() < 1e-6);
        assert!((pos.z - 5.0).abs() < 1e-6);
    }

    #[test]
    fn test_solve_moving_away() {
        let traj = |t: f64| Vec3::new(0.0, 0.0, 0.5 * t);
        let result = solve(Vec3::default(), 10.0, &traj);
        let (t_emit, _) = result.unwrap();
        let want = 10.0 / 1.5;
        assert!((t_emit - want).abs() < 1e-6);
    }
}
