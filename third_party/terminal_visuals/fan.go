package visual

import (
	"math"
	"time"
)

const FanCloseDuration = 430 * time.Millisecond

// FanFrame sequences a connector and a staggered reveal from the center row.
type FanFrame struct {
	Connector float64
	Shade     float64
	rows      int
	elapsed   time.Duration
	closing   bool
}

func fanGap(rows int) time.Duration {
	return min(70*time.Millisecond, 420*time.Millisecond/time.Duration(max(rows/2, 1)))
}

func FanOpenDuration(rows int) time.Duration {
	return 250*time.Millisecond + time.Duration(rows/2)*fanGap(rows)
}

// FanAt calculates a snapshot of opening or closing a menu fan.
func FanAt(rows int, elapsed time.Duration, closing bool) FanFrame {
	f := FanFrame{rows: rows, elapsed: elapsed, closing: closing}
	if closing {
		f.Connector = 1 - fanFraction(elapsed-180*time.Millisecond, 130*time.Millisecond)
		f.Shade = 1 - fanFraction(elapsed-310*time.Millisecond, 120*time.Millisecond)
	} else {
		f.Connector = fanEase(elapsed, 140*time.Millisecond)
		f.Shade = fanEase(elapsed, FanOpenDuration(rows))
	}
	return f
}

// Row returns the visibility of a row. The middle opens first and closes last.
func (f FanFrame) Row(index int) float64 {
	return f.Ring(index - (f.rows-1)/2)
}

// Ring returns the brightness at a signed distance from the selected row.
func (f FanFrame) Ring(distance int) float64 {
	if distance < 0 {
		distance = -distance
	}
	level := max(0, 1-.7*float64(distance)/float64(max(f.rows/2, 3)))
	if f.closing {
		return max(0, level-fanFraction(f.elapsed, 180*time.Millisecond))
	}
	return level * fanEase(f.elapsed-140*time.Millisecond-time.Duration(distance)*fanGap(f.rows), 110*time.Millisecond)
}

func fanEase(elapsed, duration time.Duration) float64 {
	p := fanFraction(elapsed, duration)
	return p * (2 - p)
}

func fanFraction(elapsed, duration time.Duration) float64 {
	return min(max(float64(elapsed)/float64(duration), 0), 1)
}

// FanRect is a region in terminal cells.
type FanRect struct{ X, Y, Width, Height int }

type FanPosition struct{ Index, X, Y, Distance int }

// SideFan lays out a right-opening fan from the middle of the left edge.
// A label's row follows the selection, and its horizontal inset follows an arc.
func SideFan(area FanRect, count, selected int) []FanPosition {
	if count <= 0 || area.Width < 12 || area.Height < 1 {
		return nil
	}
	selected = min(max(selected, 0), count-1)
	half := min(count/2, (area.Height-1)/2)
	radius := float64(half + 1)
	center := area.Y + area.Height/2
	positions := make([]FanPosition, 0, 2*half+1)
	for i := max(selected-half, 0); i <= min(selected+half, count-1); i++ {
		d := i - selected
		offset := int(math.Round(2 * (radius - math.Sqrt(radius*radius-float64(d*d)))))
		positions = append(positions, FanPosition{Index: i, X: area.X + 9 - offset, Y: center + d, Distance: d})
	}
	return positions
}

// Feather returns a falloff by distance to the nearest region. Two columns
// count as one row to account for the aspect ratio of terminal cells.
func Feather(x, y int, areas []FanRect, radius float64) float64 {
	if radius <= 0 {
		return 0
	}
	nearest := radius
	for _, area := range areas {
		if area.Width <= 0 || area.Height <= 0 {
			continue
		}
		dx := float64(max(area.X-x, x-(area.X+area.Width-1), 0)) / 2
		dy := float64(max(area.Y-y, y-(area.Y+area.Height-1), 0))
		nearest = min(nearest, math.Hypot(dx, dy))
	}
	return 1 - nearest/radius
}
