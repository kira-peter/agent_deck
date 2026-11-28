package codexchat

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
)

func TestTokenBucketsAccumulateRunningTotalsOnce(t *testing.T) {
	start := time.Date(2026, 11, 28, 12, 30, 0, 0, time.UTC)
	var history tokenHistory
	for _, sample := range []core.CodexTokenSample{
		{At: start, Input: 100, Output: 10},
		{At: start.Add(800 * time.Millisecond), Input: 160, Output: 20},
		{At: start.Add(time.Second), Input: 160, Output: 20},     // duplicate
		{At: start.Add(2 * time.Second), Input: 120, Output: 20}, // stale update
		{At: start.Add(3 * time.Second), Input: 200, Output: 27},
	} {
		history.record(sample)
	}
	buckets, origin, step := history.buckets(8, start.Add(4*time.Second))
	if !origin.Equal(start) || step != time.Second || len(buckets) != 5 {
		t.Fatalf("unexpected time axis: %v %v %v", origin, step, buckets)
	}
	want := []tokenBucket{{160, 20, 180}, {0, 0, 180}, {0, 0, 180}, {40, 7, 227}, {0, 0, 227}}
	for i, b := range buckets {
		if b != want[i] {
			t.Errorf("bucket %d: got %+v, want %+v", i, b, want[i])
		}
	}
	compressed, _, step := history.buckets(2, start.Add(4*time.Second))
	if len(compressed) > 2 || step <= time.Second || compressed[len(compressed)-1].total != 227 {
		t.Fatalf("unexpected compressed buckets: %v %v", step, compressed)
	}
}

func TestTokenNotificationDrawsLineAndStackedColors(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(90, 30)
	c := chat{screen: screen, project: "demo", threadID: "active"}
	c.receive(rpcMessage{Method: "thread/tokenUsage/updated", Params: json.RawMessage(`{
		"threadId":"other","tokenUsage":{"total":{"inputTokens":800,"outputTokens":800}}
	}`)})
	c.receive(rpcMessage{Method: "thread/tokenUsage/updated", Params: json.RawMessage(`{
		"threadId":"active","tokenUsage":{"total":{"inputTokens":100,"cachedInputTokens":90,"outputTokens":35}}
	}`)})
	if len(c.chart.samples) != 1 || c.chart.samples[0].Input != 100 || c.chart.samples[0].Output != 35 {
		t.Fatalf("wrong token totals: %+v", c.chart.samples)
	}
	c.render()
	var line, inputs, outputs, overlap bool
	cells, _, _ := screen.GetContents()
	for _, cell := range cells {
		if len(cell.Runes) == 0 {
			continue
		}
		fg, bg, _ := cell.Style.Decompose()
		r := cell.Runes[0]
		isLine := r > '⠀' && r <= '⣿' && fg == totalColor
		line = line || isLine
		overlap = overlap || (isLine && (bg == inputColor || bg == outputColor))
		if r == '█' || r == '▄' {
			inputs = inputs || fg == inputColor || bg == inputColor
			outputs = outputs || fg == outputColor || bg == outputColor
		}
	}
	if !line || !inputs || !outputs || !overlap {
		t.Fatalf("missing graph layers, line=%t input=%t output=%t overlap=%t", line, inputs, outputs, overlap)
	}
	_, _, totalAxis, _ := screen.GetContent(1, 6)
	if fg, _, _ := totalAxis.Decompose(); fg != totalColor {
		t.Error("cumulative axis should match the line color")
	}
	_, _, barAxis, _ := screen.GetContent(8, 6)
	if fg, _, _ := barAxis.Decompose(); fg != inputColor {
		t.Error("bar axis should match the input color")
	}
	f3 := tcell.NewEventKey(tcell.KeyF3, 0, 0)
	c.handleKey(f3)
	c.render()
	if r, _, _, _ := screen.GetContent(2, 3); r != '▸' {
		t.Fatalf("expected collapsed token row, got %q", r)
	}
	if r, _, _, _ := screen.GetContent(chartLeft-1, 6); r == '│' {
		t.Error("the collapsed chart still occupies plot rows")
	}
	c.handleKey(f3)
	c.render()
	if r, _, _, _ := screen.GetContent(chartLeft-1, 6); r != '│' {
		t.Error("the chart axis did not return after expanding")
	}
}
