package render

import (
	"image"
	"image/color"
	"runtime"
	"sync"

	"sif/gogs/eric/relray/pkg/camera"
	"sif/gogs/eric/relray/pkg/scene"
	"sif/gogs/eric/relray/pkg/spectrum"
)

// Config controls rendering parameters.
type Config struct {
	Width, Height int
	MaxDepth      int // max ray bounces
	SamplesPerPx  int // antialiasing samples per pixel
	NumWorkers    int // 0 = runtime.NumCPU()
}

const tileSize = 32

type tile struct{ x0, y0, x1, y1 int }

// RenderFrame produces a single frame using tile-based parallel rendering.
func RenderFrame(cfg Config, sc *scene.Scene, cam *camera.Camera) *image.RGBA {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU()
	}
	if cfg.SamplesPerPx <= 0 {
		cfg.SamplesPerPx = 1
	}
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = 4
	}

	cam.Init()
	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))

	// Build tile list
	var tiles []tile
	for y := 0; y < cfg.Height; y += tileSize {
		for x := 0; x < cfg.Width; x += tileSize {
			t := tile{x0: x, y0: y, x1: x + tileSize, y1: y + tileSize}
			if t.x1 > cfg.Width {
				t.x1 = cfg.Width
			}
			if t.y1 > cfg.Height {
				t.y1 = cfg.Height
			}
			tiles = append(tiles, t)
		}
	}

	// Dispatch tiles to workers
	work := make(chan tile, len(tiles))
	for _, t := range tiles {
		work <- t
	}
	close(work)

	var wg sync.WaitGroup
	for range cfg.NumWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracer := &Tracer{Scene: sc, Camera: cam, MaxDepth: cfg.MaxDepth}
			for t := range work {
				renderTile(tracer, cfg, t, img)
			}
		}()
	}
	wg.Wait()
	return img
}

func renderTile(tracer *Tracer, cfg Config, t tile, img *image.RGBA) {
	invW := 1.0 / float64(cfg.Width)
	invH := 1.0 / float64(cfg.Height)
	invS := 1.0 / float64(cfg.SamplesPerPx)

	for y := t.y0; y < t.y1; y++ {
		for x := t.x0; x < t.x1; x++ {
			var acc spectrum.SPD
			for s := range cfg.SamplesPerPx {
				// Jitter within pixel for antialiasing (centered for single sample)
				var jx, jy float64
				if cfg.SamplesPerPx > 1 {
					jx = (float64(s) + 0.5) / float64(cfg.SamplesPerPx)
					jy = (float64(s) + 0.5) / float64(cfg.SamplesPerPx)
				} else {
					jx, jy = 0.5, 0.5
				}
				u := (float64(x) + jx) * invW
				v := (float64(y) + jy) * invH
				// Flip v so that y=0 is bottom of image
				v = 1.0 - v
				spd := tracer.Trace(u, v)
				acc = acc.Add(spd)
			}
			acc = acc.Scale(invS)

			// Convert SPD → XYZ → sRGB
			cx, cy, cz := acc.ToXYZ()
			r, g, b := spectrum.XYZToSRGB(cx, cy, cz)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}
