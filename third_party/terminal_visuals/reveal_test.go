package visual

import "testing"

func TestSplitLinesFromCenterToEdges(t *testing.T) {
	for _, tc := range []struct {
		openness    float64
		top, bottom int
	}{{0, 4, 4}, {.5, 2, 7}, {1, 0, 9}} {
		if top, bottom := SplitLines(10, tc.openness); top != tc.top || bottom != tc.bottom {
			t.Errorf("openness %v: got %d,%d", tc.openness, top, bottom)
		}
	}
}
