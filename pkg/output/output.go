package output

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
)

// SavePNG writes an image to a PNG file.
func SavePNG(filename string, img image.Image) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// AssembleVideo invokes ffmpeg to combine numbered PNGs into an MP4 video.
// pattern should be an ffmpeg-style pattern like "/tmp/frames/frame_%04d.png".
func AssembleVideo(pattern string, fps int, outPath string) error {
	cmd := exec.Command("ffmpeg",
		"-y",
		"-framerate", fmt.Sprintf("%d", fps),
		"-i", pattern,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		outPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
