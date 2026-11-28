package visual

import "math"

// TweenCell interpolates between two cell coordinates.
func TweenCell(from, to int, progress float64) int {
	p := min(max(progress, 0), 1)
	return from + int(math.Round(float64(to-from)*p))
}

// SplitLines gives the two inclusive line positions as they travel between
// the center and edges of a region. The cells strictly between them are revealed.
func SplitLines(height int, openness float64) (top, bottom int) {
	if height <= 1 {
		return 0, 0
	}
	center := (height - 1) / 2
	return TweenCell(center, 0, openness), TweenCell(center, height-1, openness)
}
