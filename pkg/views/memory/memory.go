package memory

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
	"github.com/kira-peter/agent_deck/pkg/views"
)

// MemoryView lists every memory file Claude reads and shows its content.
type MemoryView struct {
	screen       tcell.Screen
	memories     []core.Memory
	claudeDir    *core.ClaudeDir
	width        int
	height       int
	selectedIdx  int
	scrollOffset int

	// Detail mode
	detailMode   bool
	detailLines  []string
	detailScroll int

	statusMsg string
	statusErr bool
}

func NewMemoryView() views.View {
	config := core.GetConfig()
	return &MemoryView{
		claudeDir: core.NewClaudeDir(config.GetClaudeDir()),
	}
}

func (v *MemoryView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "memory",
		NameEn: "Memory",
		NameZh: "记忆",
		DescEn: "View and manage memory files",
		DescZh: "查看和管理记忆文件",
		Icon:   "🧠",
	}
}

func (v *MemoryView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()
	return v.reload()
}

// reload re-reads the memory files. Order: user file first, then by project,
// index before topics, so related files stay together.
func (v *MemoryView) reload() error {
	memories, err := v.claudeDir.GetMemories()
	if err != nil {
		return err
	}
	sort.SliceStable(memories, func(i, j int) bool {
		a, b := memories[i], memories[j]
		if (a.Kind == core.MemoryUser) != (b.Kind == core.MemoryUser) {
			return a.Kind == core.MemoryUser
		}
		if a.ProjectPath != b.ProjectPath {
			return a.ProjectPath < b.ProjectPath
		}
		if kindOrder(a.Kind) != kindOrder(b.Kind) {
			return kindOrder(a.Kind) < kindOrder(b.Kind)
		}
		return a.Name < b.Name
	})
	v.memories = memories
	if v.selectedIdx >= len(memories) {
		v.selectedIdx = max(len(memories)-1, 0)
	}
	return nil
}

func kindOrder(k core.MemoryKind) int {
	switch k {
	case core.MemoryProject:
		return 0
	case core.MemoryAuto:
		return 1
	}
	return 2
}

func kindLabel(k core.MemoryKind) string {
	switch k {
	case core.MemoryUser:
		return i18n.MemKindUser.Get()
	case core.MemoryProject:
		return i18n.MemKindProject.Get()
	case core.MemoryAuto:
		return i18n.MemKindAuto.Get()
	}
	return i18n.MemKindTopic.Get()
}

func kindColor(k core.MemoryKind) tcell.Color {
	switch k {
	case core.MemoryUser:
		return tcell.ColorYellow
	case core.MemoryProject:
		return tcell.ColorLightGreen
	case core.MemoryAuto:
		return tcell.ColorLightCyan
	}
	return tcell.ColorGray
}

func (v *MemoryView) Render() error {
	v.screen.Clear()

	if v.detailMode {
		v.renderDetail()
	} else {
		v.renderList()
	}

	v.drawHorizontalLine(v.height - 3)
	hint := i18n.MemHintList.Get()
	if v.detailMode {
		hint = i18n.MemHintDetail.Get()
	}
	v.drawCenteredText(v.height-2, hint, tcell.StyleDefault.Foreground(tcell.ColorGray))

	if v.statusMsg != "" {
		color := tcell.ColorLightGreen
		if v.statusErr {
			color = tcell.ColorRed
		}
		v.drawCenteredText(v.height-4, textwidth.Truncate(v.statusMsg, v.width-6),
			tcell.StyleDefault.Foreground(color))
	}

	v.screen.Show()
	return nil
}

// ── List ───────────────────────────────────────────────────────────────

// Each entry takes two rows: title line and a gray summary line.
const rowsPerEntry = 2

func (v *MemoryView) renderList() {
	title := fmt.Sprintf("%s (%d)", i18n.ViewMemory.Get(), len(v.memories))
	v.drawCenteredText(1, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(3)

	const startY, startX = 5, 4
	if len(v.memories) == 0 {
		v.drawCenteredText(startY+2, i18n.StatusNoData.Get(),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
		v.drawCenteredText(startY+4, textwidth.Truncate(i18n.MemEmptyHelp.Get(), v.width-8),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
		return
	}

	visible := max((v.height-startY-5)/rowsPerEntry, 1)
	if v.selectedIdx < v.scrollOffset {
		v.scrollOffset = v.selectedIdx
	}
	if v.selectedIdx >= v.scrollOffset+visible {
		v.scrollOffset = v.selectedIdx - visible + 1
	}

	innerW := v.width - startX*2
	for i := v.scrollOffset; i < len(v.memories) && i < v.scrollOffset+visible; i++ {
		m := v.memories[i]
		y := startY + (i-v.scrollOffset)*rowsPerEntry

		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if i == v.selectedIdx {
			style = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGreen)
			v.fillLine(startX-1, y, innerW+2, style)
		}

		tag := "[" + kindLabel(m.Kind) + "]"
		tagStyle := tcell.StyleDefault.Foreground(kindColor(m.Kind))
		if i == v.selectedIdx {
			tagStyle = style
		}
		v.drawText(startX, y, tag, tagStyle)

		where := m.ProjectPath
		if m.Kind == core.MemoryUser {
			where = i18n.MemUserGlobal.Get()
		}
		name := m.Name
		if m.Kind == core.MemoryAuto || m.Kind == core.MemoryTopic {
			name = "memory/" + name
		}
		titleText := where + "  " + name
		v.drawText(startX+textwidth.StringWidth(tag)+1, y,
			textwidth.Truncate(titleText, innerW-textwidth.StringWidth(tag)-1), style)

		summary := fmt.Sprintf("  %s | %s", core.FormatSize(m.Size), core.FormatRelativeTime(m.LastModified))
		if m.EntryCount > 0 {
			summary = fmt.Sprintf("  "+i18n.MemEntries.Get()+" | %s | %s",
				m.EntryCount, core.FormatSize(m.Size), core.FormatRelativeTime(m.LastModified))
		}
		v.drawText(startX, y+1, textwidth.Truncate(summary, innerW),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
	}

	if len(v.memories) > visible {
		v.drawText(v.width-10, 3-1, fmt.Sprintf("%d/%d", v.selectedIdx+1, len(v.memories)),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

// ── Detail ─────────────────────────────────────────────────────────────

func (v *MemoryView) openDetail(m core.Memory) {
	data, err := os.ReadFile(m.FilePath)
	if err != nil {
		v.setStatus(fmt.Sprintf(i18n.MemReadFail.Get(), err), true)
		return
	}
	text := strings.ReplaceAll(strings.TrimPrefix(string(data), "\xef\xbb\xbf"), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")

	v.detailLines = wrap(text, max(v.width-8, 20))
	if len(v.detailLines) == 0 {
		v.detailLines = []string{i18n.MemEmptyFile.Get()}
	}
	v.detailScroll = 0
	v.detailMode = true
	v.setStatus("", false)
}

func (v *MemoryView) detailHeight() int { return max(v.height-9, 1) }

func (v *MemoryView) renderDetail() {
	m := v.memories[v.selectedIdx]

	title := m.Name
	if m.ProjectPath != "" {
		title = m.ProjectPath + "  " + m.Name
	}
	v.drawCenteredText(1, textwidth.Truncate(title, v.width-6), tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawCenteredText(2, textwidth.Truncate(m.FilePath, v.width-6),
		tcell.StyleDefault.Foreground(tcell.ColorGray))
	v.drawHorizontalLine(3)

	h := v.detailHeight()
	for i := 0; i < h && v.detailScroll+i < len(v.detailLines); i++ {
		v.drawText(4, 5+i, v.detailLines[v.detailScroll+i], tcell.StyleDefault.Foreground(tcell.ColorWhite))
	}

	if len(v.detailLines) > h {
		pos := fmt.Sprintf("%d-%d / %d", v.detailScroll+1, min(v.detailScroll+h, len(v.detailLines)), len(v.detailLines))
		v.drawText(v.width-textwidth.StringWidth(pos)-4, 4, pos, tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

// wrap splits text into lines of at most width display cells.
func wrap(text string, width int) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if line == "" {
			out = append(out, "")
			continue
		}
		cur, curW := []rune(nil), 0
		for _, r := range line {
			w := textwidth.StringWidth(string(r))
			if curW+w > width {
				out = append(out, string(cur))
				cur, curW = nil, 0
			}
			cur = append(cur, r)
			curW += w
		}
		out = append(out, string(cur))
	}
	if len(out) == 1 && out[0] == "" {
		return nil
	}
	return out
}

// ── Events ─────────────────────────────────────────────────────────────

func (v *MemoryView) setStatus(msg string, isErr bool) { v.statusMsg, v.statusErr = msg, isErr }

func (v *MemoryView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		if v.detailMode {
			v.handleDetailKey(ev)
			return nil
		}
		return v.handleListKey(ev)
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
		if v.detailMode {
			v.openDetail(v.memories[v.selectedIdx]) // re-wrap for the new width
		}
	}
	return nil
}

func (v *MemoryView) handleListKey(ev *tcell.EventKey) error {
	switch ev.Key() {
	case tcell.KeyEscape:
		return fmt.Errorf("back")
	case tcell.KeyUp:
		v.moveSelection(-1)
	case tcell.KeyDown:
		v.moveSelection(1)
	case tcell.KeyEnter:
		if len(v.memories) > 0 {
			v.openDetail(v.memories[v.selectedIdx])
		}
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q', 'Q':
			return fmt.Errorf("quit")
		case 'k':
			v.moveSelection(-1)
		case 'j':
			v.moveSelection(1)
		case 'o', 'O':
			v.openExternal()
		case 'r', 'R':
			if err := v.reload(); err != nil {
				v.setStatus(err.Error(), true)
			} else {
				v.setStatus("", false)
			}
		}
	}
	return nil
}

func (v *MemoryView) handleDetailKey(ev *tcell.EventKey) {
	maxScroll := max(len(v.detailLines)-v.detailHeight(), 0)
	page := v.detailHeight() - 1

	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyBackspace, tcell.KeyBackspace2:
		v.detailMode = false
	case tcell.KeyUp:
		v.detailScroll--
	case tcell.KeyDown:
		v.detailScroll++
	case tcell.KeyPgUp:
		v.detailScroll -= page
	case tcell.KeyPgDn:
		v.detailScroll += page
	case tcell.KeyHome:
		v.detailScroll = 0
	case tcell.KeyEnd:
		v.detailScroll = maxScroll
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			v.detailScroll--
		case 'j':
			v.detailScroll++
		case 'g':
			v.detailScroll = 0
		case 'G':
			v.detailScroll = maxScroll
		case 'o', 'O':
			v.openExternal()
		}
	}
	v.detailScroll = min(max(v.detailScroll, 0), maxScroll)
}

func (v *MemoryView) moveSelection(delta int) {
	v.selectedIdx = min(max(v.selectedIdx+delta, 0), max(len(v.memories)-1, 0))
}

func (v *MemoryView) openExternal() {
	if len(v.memories) == 0 {
		return
	}
	path := v.memories[v.selectedIdx].FilePath
	if err := core.OpenPath(path); err != nil {
		v.setStatus(fmt.Sprintf(i18n.MemOpenFail.Get(), err), true)
		return
	}
	v.setStatus(fmt.Sprintf(i18n.MemOpened.Get(), path), false)
}

func (v *MemoryView) Cleanup() error {
	return nil
}

func (v *MemoryView) fillLine(x, y, w int, style tcell.Style) {
	for i := 0; i < w; i++ {
		v.screen.SetContent(x+i, y, ' ', nil, style)
	}
}

func (v *MemoryView) drawText(x, y int, text string, style tcell.Style) {
	textwidth.Draw(v.screen, x, y, text, style)
}

func (v *MemoryView) drawCenteredText(y int, text string, style tcell.Style) {
	textwidth.DrawCentered(v.screen, v.width, y, text, style)
}

func (v *MemoryView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
