// Package visual provides deterministic colors and effects for terminal UIs.
// Applications supply the time, cell bounds, and renderer.
package visual

import (
	"math"
	"math/rand/v2"
	"time"
)

// RGB represents an sRGB color as 0xRRGGBB.
type RGB uint32

// Mix interpolates color channels independently, clamping amount to [0, 1].
func Mix(from, to RGB, amount float64) RGB {
	if math.IsNaN(amount) || amount <= 0 {
		return from
	}
	if amount >= 1 {
		return to
	}
	var color RGB
	for _, shift := range [...]uint{16, 8, 0} {
		a := float64(from >> shift & 255)
		b := float64(to >> shift & 255)
		color |= RGB(math.Round(a+(b-a)*amount)) << shift
	}
	return color
}

// Sweep defines a light band that traverses a line once, or loops with a pause.
type Sweep struct {
	Duration time.Duration
	Radius   float64
}

// Band is the light profile at a particular instant.
type Band struct {
	center float64
	radius float64
}

// At returns the band for a single pass. A zero Band draws no highlight.
func (s Sweep) At(elapsed time.Duration, width int) Band {
	if elapsed < 0 || elapsed >= s.Duration || width <= 0 || s.Radius <= 0 {
		return Band{}
	}
	progress := float64(elapsed) / float64(s.Duration)
	return Band{center: -s.Radius + progress*(float64(width-1)+2*s.Radius), radius: s.Radius}
}

// Loop repeats a sweep with a rest between passes.
func (s Sweep) Loop(elapsed, pause time.Duration, width int) Band {
	if s.Duration <= 0 || elapsed < 0 || pause < 0 {
		return Band{}
	}
	return s.At(elapsed%(s.Duration+pause), width)
}

// Strength gives the smooth highlight intensity of a cell in [0, 1].
func (b Band) Strength(cell int) float64 {
	if b.radius == 0 {
		return 0
	}
	x := 1 - math.Abs(float64(cell)-b.center)/b.radius
	if x <= 0 {
		return 0
	}
	return x * x * (3 - 2*x)
}

// EaseInOutCubic maps progress to a curve with smooth start and end points.
func EaseInOutCubic(progress float64) float64 {
	p := min(max(progress, 0), 1)
	if p < .5 {
		return 4 * p * p * p
	}
	left := 2*p - 2
	return 1 + left*left*left/2
}

// Speck is a cell relative to a field's origin. Brightness is in [0, 1].
type Speck struct {
	X, Y       int
	Brightness float64
}

// Starfield visits a sparse pattern of drifting cells. Pass elapsed=0 for a
// static frame. A stable seed keeps the pattern consistent between renders.
func Starfield(width, height, count int, seed uint64, elapsed time.Duration, visit func(Speck)) {
	if width <= 0 || height <= 0 || count <= 0 || visit == nil {
		return
	}
	steps := int(max(elapsed, 0) / (600 * time.Millisecond))
	rng := rand.New(rand.NewPCG(seed, 0x8a5c1))
	for range count {
		x, y := rng.IntN(width), rng.IntN(height)
		period := 2 + rng.IntN(4)
		visit(Speck{
			X:          (x + steps/period) % width,
			Y:          (y + steps/(period+3)) % height,
			Brightness: 0.22 + rng.Float64()*0.60,
		})
	}
}
