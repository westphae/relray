package output

import (
	"image"
	"image/png"
	"os"
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
