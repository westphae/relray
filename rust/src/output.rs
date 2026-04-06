use image::RgbaImage;
use std::process::Command;

/// Save an image as a PNG file.
pub fn save_png(filename: &str, img: &RgbaImage) -> Result<(), Box<dyn std::error::Error>> {
    img.save(filename)?;
    Ok(())
}

/// Assemble numbered PNG frames into an MP4 video using ffmpeg.
pub fn assemble_video(pattern: &str, fps: u32, out_path: &str) -> Result<(), Box<dyn std::error::Error>> {
    let status = Command::new("ffmpeg")
        .args([
            "-y",
            "-framerate", &fps.to_string(),
            "-i", pattern,
            "-c:v", "libx264",
            "-pix_fmt", "yuv420p",
            out_path,
        ])
        .status()?;

    if !status.success() {
        return Err(format!("ffmpeg exited with status {}", status).into());
    }
    Ok(())
}
