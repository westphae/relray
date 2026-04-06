use crate::vec::Vec3;

/// Returns the Lorentz factor for velocity beta (in units of c).
#[allow(dead_code)]
pub fn gamma(beta: Vec3) -> f64 {
    let b2 = beta.length_sq();
    if b2 == 0.0 {
        return 1.0;
    }
    1.0 / (1.0 - b2).sqrt()
}

/// Result of a relativistic aberration computation.
pub struct AberrationResult {
    /// Unit ray direction in world frame (camera -> scene).
    pub dir: Vec3,
    /// Frequency ratio f_obs / f_emit (>1 = blueshift).
    pub doppler: f64,
}

/// Transforms a ray direction from the observer's rest frame to the world frame,
/// and simultaneously computes the Doppler factor.
///
/// `dir_obs` is the ray direction (camera -> scene) in the observer's rest frame.
/// `beta` is the observer's velocity as a fraction of c in the world frame.
///
/// Method: the photon arriving at the camera propagates in direction -dir_obs.
/// Construct the photon's null 4-wavevector k_obs = (1, -dir_obs) in the observer
/// frame, then apply the Lorentz boost (+beta) to get k_world. The world-frame
/// ray direction is the negation of the spatial part of k_world (normalized).
/// The Doppler factor is f_obs/f_emit = 1/k_world^0.
pub fn aberrate(dir_obs: Vec3, beta: Vec3) -> AberrationResult {
    let b2 = beta.length_sq();
    if b2 == 0.0 {
        return AberrationResult {
            dir: dir_obs,
            doppler: 1.0,
        };
    }

    let g = 1.0 / (1.0 - b2).sqrt();

    // Photon propagation direction in observer frame
    let p = -dir_obs;

    // Null 4-wavevector in observer frame: k_obs = (1, p)
    // Lorentz boost to world frame (boost velocity = +beta):
    //   k_world^0 = gamma * (1 + beta . p)
    //   k_world_spatial = p + beta*gamma + (gamma-1)/b2 * (beta.p) * beta
    let bdotp = beta.dot(p);
    let kw0 = g * (1.0 + bdotp);
    let factor = (g - 1.0) / b2 * bdotp;
    let kw_spatial = p + beta * g + beta * factor;

    // Ray direction = negated photon propagation direction
    let dir = (-kw_spatial).normalize();
    let doppler = 1.0 / kw0;

    AberrationResult { dir, doppler }
}

#[cfg(test)]
mod tests {
    use super::*;

    const EPS: f64 = 1e-9;

    fn assert_close(label: &str, got: f64, want: f64) {
        assert!(
            (got - want).abs() < EPS,
            "{label}: got {got}, want {want}"
        );
    }

    #[test]
    fn test_gamma() {
        assert_close("beta=0", gamma(Vec3::default()), 1.0);
        assert_close(
            "beta=0.5",
            gamma(Vec3::new(0.0, 0.0, 0.5)),
            1.0 / 0.75_f64.sqrt(),
        );
        assert_close(
            "beta=0.9",
            gamma(Vec3::new(0.9, 0.0, 0.0)),
            1.0 / (1.0 - 0.81_f64).sqrt(),
        );
    }

    #[test]
    fn test_aberrate_forward() {
        let beta = Vec3::new(0.0, 0.0, 0.5);
        let r = aberrate(Vec3::new(0.0, 0.0, 1.0), beta);
        assert_close("forward dir.X", r.dir.x, 0.0);
        assert_close("forward dir.Y", r.dir.y, 0.0);
        assert_close("forward dir.Z", r.dir.z, 1.0);
        // Looking forward while moving forward = blueshift
        // D = sqrt((1+beta)/(1-beta)) = sqrt(3)
        assert_close("forward Doppler", r.doppler, 3.0_f64.sqrt());
    }

    #[test]
    fn test_aberrate_backward() {
        let beta = Vec3::new(0.0, 0.0, 0.5);
        let r = aberrate(Vec3::new(0.0, 0.0, -1.0), beta);
        assert_close("backward dir.Z", r.dir.z, -1.0);
        // Looking backward = receding = redshift
        assert_close("backward Doppler", r.doppler, 1.0 / 3.0_f64.sqrt());
    }

    #[test]
    fn test_aberrate_sideways() {
        let beta = Vec3::new(0.0, 0.0, 0.5);
        let r = aberrate(Vec3::new(1.0, 0.0, 0.0), beta);

        // Transverse Doppler = 1/gamma (always a redshift)
        let g = gamma(beta);
        assert_close("sideways Doppler", r.doppler, 1.0 / g);

        // A sideways ray in the observer frame maps to a MORE BACKWARD direction
        // in the world frame.
        assert!(
            r.dir.z < 0.0,
            "expected negative Z for sideways observer ray, got {}",
            r.dir.z
        );
        assert!(
            r.dir.x > 0.0,
            "expected positive X preserved, got {}",
            r.dir.x
        );
        assert_close("unit length", r.dir.length(), 1.0);
    }

    #[test]
    fn test_aberrate_zero_beta() {
        let d = Vec3::new(0.5, 0.5, (0.5_f64).sqrt()).normalize();
        let r = aberrate(d, Vec3::default());
        assert_close("Doppler", r.doppler, 1.0);
        assert_close("dir.X", r.dir.x, d.x);
        assert_close("dir.Y", r.dir.y, d.y);
        assert_close("dir.Z", r.dir.z, d.z);
    }

    #[test]
    fn test_aberrate_high_beta() {
        let beta = Vec3::new(0.0, 0.0, 0.9);
        let dir_obs = Vec3::new(1.0, 0.0, -0.1).normalize();
        let r = aberrate(dir_obs, beta);
        assert!(
            r.dir.z < dir_obs.z,
            "at beta=0.9, expected world dir more backward, got dir.Z={}",
            r.dir.z
        );
    }

    #[test]
    fn test_aberrate_doppler_symmetry() {
        // D(forward) * D(backward) = 1
        let beta = Vec3::new(0.0, 0.0, 0.5);
        let rf = aberrate(Vec3::new(0.0, 0.0, 1.0), beta);
        let rb = aberrate(Vec3::new(0.0, 0.0, -1.0), beta);
        assert_close("D_fwd * D_bwd", rf.doppler * rb.doppler, 1.0);
    }

    #[test]
    fn test_aberrate_round_trip() {
        let beta = Vec3::new(0.2, 0.0, 0.3);
        let dir_obs = Vec3::new(0.5, 0.7, -0.3).normalize();

        let r1 = aberrate(dir_obs, beta);
        let r2 = aberrate(r1.dir, -beta);

        assert_close("round-trip X", r2.dir.x, dir_obs.x);
        assert_close("round-trip Y", r2.dir.y, dir_obs.y);
        assert_close("round-trip Z", r2.dir.z, dir_obs.z);
        assert_close("round-trip Doppler", r1.doppler * r2.doppler, 1.0);
    }
}
