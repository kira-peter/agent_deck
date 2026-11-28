package visual

import (
	"testing"
	"time"
)

func TestFanOpensCenterFirstAndRetractsLast(t *testing.T) {
	start := FanAt(3, 120*time.Millisecond, false)
	if start.Connector <= 0 || start.Row(1) != 0 {
		t.Fatal("connector should lead the reveal")
	}
	opening := FanAt(3, 180*time.Millisecond, false)
	if opening.Connector != 1 || opening.Row(1) <= 0 || opening.Row(0) != 0 {
		t.Fatal("middle row should reveal first")
	}
	closing := FanAt(3, 100*time.Millisecond, true)
	if closing.Connector != 1 || closing.Row(1) <= closing.Row(0) {
		t.Fatal("outer rows should fade before the line retracts")
	}
	done := FanAt(3, FanCloseDuration, true)
	if done.Connector != 0 || done.Row(1) != 0 {
		t.Fatal("the fan should fully close")
	}
}

func TestSideFanIsCurvedAndShadeFeathersInCellSpace(t *testing.T) {
	area := FanRect{X: 72, Y: 3, Width: 55, Height: 20}
	rows := SideFan(area, 3, 1)
	if len(rows) != 3 || rows[0].X != rows[2].X || rows[0].X >= rows[1].X || rows[1].Y != 13 || rows[1].X < area.X {
		t.Fatalf("fan must curve symmetrically on its own pane: %+v", rows)
	}
	cover := []FanRect{{X: 82, Y: 13, Width: 10, Height: 1}}
	if Feather(82, 13, cover, 3) != 1 || Feather(80, 13, cover, 3) != Feather(82, 12, cover, 3) || Feather(76, 13, cover, 3) != 0 {
		t.Fatal("shade should fade gradually in rows and columns")
	}
}
