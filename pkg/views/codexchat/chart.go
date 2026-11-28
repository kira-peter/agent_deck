package codexchat

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
)

type tokenHistory struct {
	samples []core.CodexTokenSample
}

func (h *tokenHistory) record(sample core.CodexTokenSample) {
	sample.Input = max(sample.Input, 0)
	sample.Output = max(sample.Output, 0)
	if len(h.samples) > 0 {
		last := h.samples[len(h.samples)-1]
		if sample.At.Before(last.At) {
			sample.At = last.At
		}
		sample.Input = max(last.Input, sample.Input)
		sample.Output = max(last.Output, sample.Output)
		if sample.Input == last.Input && sample.Output == last.Output {
			return
		}
	}
	h.samples = append(h.samples, sample)
}

type tokenBucket struct {
	input, output int64
	total         int64
}

func (h tokenHistory) buckets(columns int, now time.Time) ([]tokenBucket, time.Time, time.Duration) {
	if len(h.samples) == 0 {
		return nil, now, 0
	}
	first := h.samples[0].At
	end := now
	if last := h.samples[len(h.samples)-1].At; last.After(end) {
		end = last
	}
	step := time.Second
	for end.Truncate(step).Sub(first.Truncate(step))/step >= time.Duration(columns) {
		step *= 2
	}
	start := first.Truncate(step)
	count := int(end.Truncate(step).Sub(start)/step) + 1
	result := make([]tokenBucket, count)
	var prev core.CodexTokenSample
	for _, sample := range h.samples {
		bucket := &result[int(sample.At.Sub(start)/step)]
		bucket.input += sample.Input - prev.Input
		bucket.output += sample.Output - prev.Output
		prev = sample
	}
	var total int64
	for i := range result {
		total += result[i].input + result[i].output
		result[i].total = total
	}
	return result, start, step
}

const chartLeft = 16

var (
	totalColor  = tcell.ColorLightYellow
	inputColor  = tcell.ColorDeepSkyBlue
	outputColor = tcell.ColorLightSalmon
	chartDim    = baseStyle.Foreground(inkColor(ink.Muted))
)

// drawTokenChart overlays session totals on per-bucket bars using two scales.
func (c *chat) drawTokenChart(y, rows, width int) int {
	heading := local("▾ TOKENS [F3]", "▾ Token 用量 [F3]")
	if rows < 2 {
		heading = local("▸ TOKENS", "▸ Token 用量")
	} else if c.chartCollapsed {
		heading = local("▸ TOKENS [F3]", "▸ Token 用量 [F3]")
	}
	c.draw(2, y, width-4, heading, chartDim.Bold(true))
	if len(c.chart.samples) == 0 || c.chart.samples[len(c.chart.samples)-1].Input+c.chart.samples[len(c.chart.samples)-1].Output == 0 {
		x := 4 + textwidth.StringWidth(heading)
		c.draw(x, y, width-x-2, local("Waiting for usage…", "等待用量…"), chartDim)
		return y + 1
	}
	last := c.chart.samples[len(c.chart.samples)-1]
	total := last.Input + last.Output
	if c.chartCollapsed || rows < 2 {
		x := 4 + textwidth.StringWidth(heading)
		c.draw(x, y, width-x-2, local("◆ Total ", "◆ 累计 ")+strconv.FormatInt(total, 10), baseStyle.Foreground(totalColor))
		return y + 1
	}
	legend := []struct {
		text  string
		color tcell.Color
	}{
		{local("◆ Total ", "◆ 累计 ") + strconv.FormatInt(total, 10), totalColor},
		{local("■ Input ", "■ 输入 ") + strconv.FormatInt(last.Input, 10), inputColor},
		{local("■ Output ", "■ 输出 ") + strconv.FormatInt(last.Output, 10), outputColor},
	}
	legendX := 2
	for _, label := range legend {
		c.draw(legendX, y+1, width-legendX-2, label.text, baseStyle.Foreground(label.color))
		legendX += textwidth.StringWidth(label.text) + 2
	}
	plotWidth := width - chartLeft - 2
	if plotWidth < 16 {
		return y + 2
	}
	data, start, step := c.chart.buckets(max(plotWidth/2, 2), time.Now())
	c.draw(2, y, width-4, heading+fmt.Sprintf(local("  ·  %s / bar", "  ·  每柱 %s"), step), chartDim.Bold(true))
	plotY := y + 3
	axisY := plotY + rows
	var peak int64
	for _, bucket := range data {
		peak = max(peak, bucket.input+bucket.output)
	}
	totalStyle := baseStyle.Foreground(totalColor)
	barScaleStyle := baseStyle.Foreground(inputColor)
	c.draw(1, plotY-1, 7, local("Total", "累计"), totalStyle)
	c.draw(8, plotY-1, 7, local("Bar", "单柱"), barScaleStyle)
	for _, tick := range []struct {
		y          int
		total, bar int64
	}{{plotY, total, peak}, {axisY, 0, 0}} {
		c.draw(1, tick.y, 7, axisTokens(tick.total), totalStyle)
		c.draw(8, tick.y, 7, axisTokens(tick.bar), barScaleStyle)
	}
	for row := 0; row < rows; row++ {
		c.screen.SetContent(chartLeft-1, plotY+row, '│', nil, borderStyle)
		if row == 0 || (rows >= 4 && row == rows/2) {
			c.draw(chartLeft, plotY+row, plotWidth, strings.Repeat("┄", plotWidth), borderStyle)
		}
	}
	c.draw(chartLeft-1, axisY, plotWidth+1, "└"+strings.Repeat("─", plotWidth), borderStyle)
	plot := make([][]rune, rows)
	for row := range plot {
		plot[row] = make([]rune, plotWidth)
	}
	var previousX, previousY int
	for i, bucket := range data {
		left := i * plotWidth / len(data)
		right := (i + 1) * plotWidth / len(data)
		center := (left + right - 1) / 2
		lineX := 2 * center
		level := rows*4 - 1 - int(math.Round(float64(bucket.total)/float64(total)*float64(rows*4-1)))
		if i == 0 {
			previousY = rows*4 - 1
		}
		brailleLine(plot, previousX, previousY, lineX, level)
		previousX, previousY = lineX, level
		if peak > 0 && bucket.input+bucket.output > 0 {
			barWidth := min(max(right-left-1, 1), 5)
			c.drawTokenBar(chartLeft+center-(barWidth-1)/2, axisY-1, barWidth, rows, bucket, peak)
		}
	}
	brailleLine(plot, previousX, previousY, plotWidth*2-1, previousY)
	for row, cells := range plot {
		for col, dots := range cells {
			if dots == 0 {
				continue
			}
			x, y := chartLeft+col, plotY+row
			glyph, _, style, _ := c.screen.GetContent(x, y)
			if glyph == '▄' {
				// A half block can already use both colors. An underline keeps
				// the line visible without replacing either half of the bar.
				c.screen.SetContent(x, y, glyph, nil, style.Underline(true, totalColor))
				continue
			}
			lineStyle := totalStyle.Bold(true)
			if glyph == '█' {
				// Keep the bar visible as the background wherever the line crosses it.
				fg, _, _ := style.Decompose()
				lineStyle = lineStyle.Background(fg)
			}
			c.screen.SetContent(x, y, '⠀'+dots, nil, lineStyle)
		}
	}
	previousLabel := chartLeft - 1
	for i := 0; i < 3; i++ {
		bucket := i * (len(data) - 1) / 2
		if i > 0 && bucket == (i-1)*(len(data)-1)/2 {
			continue
		}
		label := start.Add(time.Duration(bucket) * step).Local().Format("15:04:05")
		if step*time.Duration(len(data)) >= 24*time.Hour {
			label = start.Add(time.Duration(bucket) * step).Local().Format("01/02 15:04")
		}
		left := bucket * plotWidth / len(data)
		right := (bucket + 1) * plotWidth / len(data)
		x := chartLeft + min(max((left+right-len(label))/2, 0), plotWidth-len(label))
		if x <= previousLabel || (i == 1 && plotWidth < 3*(len(label)+1)) {
			continue
		}
		c.draw(x, axisY+1, width-x-2, label, chartDim)
		previousLabel = x + len(label)
	}
	c.draw(1, axisY+1, chartLeft-2, local("Time", "时间"), chartDim)
	return axisY + 2
}

func axisTokens(n int64) string {
	switch {
	case n >= 1_000_000_000_000:
		return fmt.Sprintf("%.1fT", float64(n)/1e12)
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1e9)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	default:
		return strconv.FormatInt(n, 10)
	}
}

func (c *chat) drawTokenBar(x, base, width, rows int, b tokenBucket, peak int64) {
	total := b.input + b.output
	units := max(1, int(math.Round(float64(total)/float64(peak)*float64(2*rows))))
	input := units
	if b.input == 0 {
		input = 0
	} else if b.output > 0 {
		units = max(units, 2)
		input = min(max(int(math.Round(float64(b.input)/float64(total)*float64(units))), 1), units-1)
	}
	for half := 0; half < units; half += 2 {
		bottom := outputColor
		if half < input {
			bottom = inputColor
		}
		glyph := '▄'
		style := baseStyle.Foreground(bottom)
		if half+1 < units {
			switch {
			case half+1 == input:
				style = style.Background(outputColor)
			default:
				glyph = '█'
			}
		}
		for col := 0; col < width; col++ {
			c.screen.SetContent(x+col, base-half/2, glyph, nil, style)
		}
	}
}

// brailleLine draws with two horizontal and four vertical dots per cell.
func brailleLine(plot [][]rune, x0, y0, x1, y1 int) {
	dots := [4][2]rune{{1, 8}, {2, 16}, {4, 32}, {64, 128}}
	dx, dy := x1-x0, int(math.Abs(float64(y1-y0)))
	sy := 1
	if y1 < y0 {
		sy = -1
	}
	delta := dx - dy
	for {
		plot[y0/4][x0/2] |= dots[y0%4][x0%2]
		if x0 == x1 && y0 == y1 {
			return
		}
		step := 2 * delta
		if step > -dy {
			delta -= dy
			x0++
		}
		if step < dx {
			delta += dx
			y0 += sy
		}
	}
}

func (c *chat) draw(x, y, available int, text string, style tcell.Style) {
	if available > 0 {
		textwidth.Draw(c.screen, x, y, textwidth.Truncate(text, available), style)
	}
}
