package codexchat

import (
	"slices"
	"strings"
	"time"

	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
	visual "github.com/will-to-meaning/terminal_visuals"
)

const (
	paneReflow = 220 * time.Millisecond
	paneReveal = 290 * time.Millisecond
)

type paneLayout struct {
	tabs   []*chat
	active int
	split  bool
}

type paneMotion struct {
	before  paneLayout
	started time.Time
	closing bool
}

type stagedPane struct {
	chat     *chat
	bounds   paneBounds
	framed   bool
	reveal   bool
	openness float64
}

func (w *workspace) rememberLayout() {
	if w.motionOff || w.before != nil {
		return
	}
	w.before = &paneLayout{tabs: slices.Clone(w.tabs), active: w.active, split: w.split}
}

func (v *workspaceView) startPaneMotion() {
	w := v.workspace
	if w.motionOff {
		w.before, v.motion = nil, nil
		return
	}
	if w.before != nil {
		v.motion = &paneMotion{before: *w.before, started: time.Now(), closing: len(w.before.tabs) > len(w.tabs)}
		w.before = nil
	}
	if v.motion != nil && time.Since(v.motion.started) >= paneReveal+v.motion.reflow(len(w.tabs)) {
		v.motion = nil
	}
}

func (m *paneMotion) reflow(count int) time.Duration {
	if count == 0 || len(m.before.tabs) == 0 {
		return 0
	}
	return paneReflow
}

func panesFor(area paneBounds, layout paneLayout) []stagedPane {
	if len(layout.tabs) == 0 {
		return nil
	}
	boxes, start := arrangePanes(area, len(layout.tabs), layout.active, layout.split)
	panes := make([]stagedPane, len(boxes))
	for i, b := range boxes {
		panes[i] = stagedPane{chat: layout.tabs[start+i], bounds: b, framed: len(boxes) > 1}
	}
	return panes
}

func (v *workspaceView) paneFrames(area paneBounds, focus int) ([]stagedPane, int) {
	w := v.workspace
	final := panesFor(area, paneLayout{tabs: w.tabs, active: focus, split: w.split})
	m := v.motion
	if m == nil {
		return final, len(final)
	}
	old := panesFor(area, m.before)
	elapsed := time.Since(m.started)
	reflow := m.reflow(len(w.tabs))
	if m.closing && elapsed < paneReveal {
		for i := range old {
			if !slices.Contains(w.tabs, old[i].chat) {
				old[i].reveal = true
				old[i].openness = 1 - motionProgress(elapsed, paneReveal)
			}
		}
		return old, len(final)
	}
	moveElapsed := elapsed
	if m.closing {
		moveElapsed -= paneReveal
	}
	if moveElapsed < reflow && len(old) > 0 {
		var moving []stagedPane
		for _, pane := range final {
			if index := slices.IndexFunc(old, func(prev stagedPane) bool { return prev.chat == pane.chat }); index >= 0 {
				prev := old[index]
				pane.bounds = interpolatePane(prev.bounds, pane.bounds, motionProgress(moveElapsed, reflow))
				pane.framed = prev.framed || pane.framed
				moving = append(moving, pane)
			}
		}
		if len(moving) == 0 {
			if m.closing {
				return final, len(final)
			}
			return old, len(final)
		}
		return moving, len(final)
	}
	if !m.closing {
		for i := range final {
			if !slices.Contains(m.before.tabs, final[i].chat) {
				final[i].reveal = true
				final[i].openness = motionProgress(elapsed-reflow, paneReveal)
			}
		}
	}
	return final, len(final)
}

func interpolatePane(from, to paneBounds, p float64) paneBounds {
	return paneBounds{x: visual.TweenCell(from.x, to.x, p), y: visual.TweenCell(from.y, to.y, p), width: visual.TweenCell(from.width, to.width, p), height: visual.TweenCell(from.height, to.height, p)}
}

func motionProgress(elapsed, duration time.Duration) float64 {
	return visual.EaseInOutCubic(float64(elapsed) / float64(duration))
}

func (v *workspaceView) drawSplitLines(bounds paneBounds, top, bottom int) {
	style := baseStyle.Foreground(inkColor(ink.Highlight)).Bold(true)
	line := strings.Repeat("━", bounds.width)
	textwidth.Draw(v.screen, bounds.x, top, line, style)
	if top != bottom {
		textwidth.Draw(v.screen, bounds.x, bottom, line, style)
	}
}
