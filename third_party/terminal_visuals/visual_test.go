package visual

import (
	"slices"
	"testing"
	"time"
)

func TestSweepAndBlend(t *testing.T) {
	s := Sweep{Duration: time.Second, Radius: 3}
	for _, tc := range []struct {
		at   time.Duration
		want float64
	}{{0, 0}, {500 * time.Millisecond, 1}, {time.Second, 0}} {
		if got := s.At(tc.at, 11).Strength(5); got != tc.want {
			t.Errorf("%v: got %v, want %v", tc.at, got, tc.want)
		}
	}
	if s.Loop(1500*time.Millisecond, time.Second, 11).Strength(5) != 0 || s.Loop(2500*time.Millisecond, time.Second, 11).Strength(5) != 1 {
		t.Error("loop should rest and then restart")
	}
	if got := Mix(0x102030, 0x5080a0, 0.5); got != 0x305068 {
		t.Errorf("unexpected blend: %#06x", got)
	}
	if EaseInOutCubic(-1) != 0 || EaseInOutCubic(.25) != .0625 || EaseInOutCubic(2) != 1 {
		t.Error("easing did not clamp and settle")
	}
}

func TestStarfieldIsRepeatableAndClipped(t *testing.T) {
	frame := func(at time.Duration) []Speck {
		var out []Speck
		Starfield(50, 14, 24, 17, at, func(p Speck) {
			if p.X < 0 || p.X >= 50 || p.Y < 0 || p.Y >= 14 || p.Brightness < 0 || p.Brightness > 1 {
				t.Fatalf("invalid speck: %+v", p)
			}
			out = append(out, p)
		})
		return out
	}
	a := frame(0)
	if len(a) != 24 || !slices.Equal(a, frame(0)) || slices.Equal(a, frame(10*time.Second)) {
		t.Error("stars should be stable at a given time and drift over time")
	}
}

func TestOptionListAndBlackTheme(t *testing.T) {
	var list OptionList
	list.Move(8, -1)
	if start, end := list.Visible(8, 3); list.Selected != 7 || start != 5 || end != 8 {
		t.Fatalf("wrapped list window: %d %d selected %d", start, end, list.Selected)
	}
	list.Move(8, 1)
	if start, end := list.Visible(8, 3); list.Selected != 0 || start != 0 || end != 3 {
		t.Fatalf("rewound list window: %d %d selected %d", start, end, list.Selected)
	}
	if PaletteFor("black").Background != 0 || PaletteFor("unknown") != Ink() || !Blink(320*time.Millisecond) || Blink(time.Second) {
		t.Error("invalid palette or flash timing")
	}
}
