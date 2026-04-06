use crate::vec::Vec3;

pub struct Camera {
    pub position: Vec3,
    pub look_at: Vec3,
    pub up: Vec3,
    pub vfov: f64,
    pub aspect: f64,
    pub beta: Vec3,

    // Computed by init()
    u: Vec3,
    v: Vec3,
    w: Vec3,
    half_w: f64,
    half_h: f64,
}

impl Camera {
    pub fn new(position: Vec3, look_at: Vec3, up: Vec3, vfov: f64, aspect: f64, beta: Vec3) -> Self {
        let mut cam = Camera {
            position, look_at, up, vfov, aspect, beta,
            u: Vec3::default(), v: Vec3::default(), w: Vec3::default(),
            half_w: 0.0, half_h: 0.0,
        };
        cam.init();
        cam
    }

    fn init(&mut self) {
        let theta = self.vfov.to_radians();
        self.half_h = (theta / 2.0).tan();
        self.half_w = self.half_h * self.aspect;

        self.w = (self.position - self.look_at).normalize();
        self.u = self.up.cross(self.w).normalize();
        self.v = self.w.cross(self.u);
    }

    /// Returns the ray direction in the observer's rest frame for
    /// normalized screen coordinates (s, t) where (0,0) is bottom-left.
    pub fn ray_dir(&self, s: f64, t: f64) -> Vec3 {
        let x = (2.0 * s - 1.0) * self.half_w;
        let y = (2.0 * t - 1.0) * self.half_h;
        (self.u * x + self.v * y - self.w).normalize()
    }
}
