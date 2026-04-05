package render

import (
	"image"
	"image/color"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"

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

	// Dispatch tiles to workers via atomic counter (avoids channel overhead)
	var nextTile atomic.Int64
	var wg sync.WaitGroup
	for i := range cfg.NumWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(i) * 31337))
			tracer := &Tracer{Scene: sc, Camera: cam, MaxDepth: cfg.MaxDepth, Rng: rng}
			for {
				idx := int(nextTile.Add(1)) - 1
				if idx >= len(tiles) {
					break
				}
				renderTile(tracer, cfg, tiles[idx], img, rng)
			}
		}()
	}
	wg.Wait()
	return img
}

func renderTile(tracer *Tracer, cfg Config, t tile, img *image.RGBA, rng *rand.Rand) {
	invW := 1.0 / float64(cfg.Width)
	invH := 1.0 / float64(cfg.Height)
	invS := 1.0 / float64(cfg.SamplesPerPx)

	for y := t.y0; y < t.y1; y++ {
		for x := t.x0; x < t.x1; x++ {
			var acc spectrum.SPD
			for range cfg.SamplesPerPx {
				jx := rng.Float64()
				jy := rng.Float64()
				u := (float64(x) + jx) * invW
				v := 1.0 - (float64(y)+jy)*invH
				spd := tracer.Trace(u, v)
				acc.AddInPlace(&spd)
			}
			acc.ScaleInPlace(invS)

			cx, cy, cz := acc.ToXYZ()
			r, g, b := spectrum.XYZToSRGB(cx, cy, cz)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}
