package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	rowSizeColW  = 10 // fits "999.9 GiB"/"1023.9 MiB" without overflow
	rowMtimeColW = 11
	rowMtimeFmt  = "01-02 15:04"
)

var (
	cursorFg = lipgloss.AdaptiveColor{Light: "6", Dark: "14"}   // cyan / bright cyan
	dirFg    = lipgloss.AdaptiveColor{Light: "4", Dark: "12"}   // blue / bright blue
	dimFg    = lipgloss.AdaptiveColor{Light: "7", Dark: "8"}    // light gray / dark gray
	errFg    = lipgloss.AdaptiveColor{Light: "1", Dark: "9"}    // red / bright red
	accentFg = lipgloss.AdaptiveColor{Light: "5", Dark: "13"}   // magenta / bright magenta
	borderFg = lipgloss.AdaptiveColor{Light: "6", Dark: "14"}   // cyan / bright cyan

	rowCursorStyle = lipgloss.NewStyle().Bold(true).Foreground(cursorFg)
	rowDirStyle    = lipgloss.NewStyle().Foreground(dirFg)
	rowFileStyle   = lipgloss.NewStyle()
)

const (
	cursorPrefix = "▶ "
	noCursorPad  = "  "
)

type itemDelegate struct{}

func (itemDelegate) Height() int                             { return 1 }
func (itemDelegate) Spacing() int                            { return 0 }
func (itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(item)
	if !ok {
		return
	}

	icon := iconFor(it.info)
	mtime := it.info.ModTime.Format(rowMtimeFmt)

	var size string
	if it.info.IsDir {
		size = strings.Repeat(" ", rowSizeColW)
	} else {
		size = humanSizeRight(it.info.Size, rowSizeColW)
	}

	const sizeMtimeGap = "   " // 3 spaces between size col and mtime col
	const cursorColW = 2       // "▶ " or "  " prefix
	avail := m.Width() - cursorColW - visualWidth(icon) - 1 - rowSizeColW - len(sizeMtimeGap) - rowMtimeColW - 1
	if avail < 4 {
		avail = 4
	}

	name := truncateRight(it.info.Name, avail)
	name = padRight(name, avail)

	prefix := noCursorPad
	isCursor := index == m.Index()
	if isCursor {
		prefix = cursorPrefix
	}

	body := fmt.Sprintf("%s%s %s %s%s%s", prefix, icon, name, size, sizeMtimeGap, mtime)

	switch {
	case isCursor:
		body = rowCursorStyle.Render(body)
	case it.info.IsDir:
		body = rowDirStyle.Render(body)
	default:
		body = rowFileStyle.Render(body)
	}

	_, _ = fmt.Fprint(w, body)
}

// visualWidth returns the on-screen cell width of s. ASCII is 1 cell; a
// small allow-list of common narrow non-ASCII runes (arrows, box-drawing) is
// treated as 1 cell; everything else non-ASCII is treated as 2 (matches the
// emoji icons we use). Returns at least 1 for non-empty input.
func visualWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case r <= 0x7f:
			w++
		case isNarrowRune(r):
			w++
		default:
			w += 2
		}
	}
	if w == 0 {
		w = 1
	}
	return w
}

// isNarrowRune reports whether r is a non-ASCII rune that terminals
// typically render as a single cell.
func isNarrowRune(r rune) bool {
	switch r {
	case '↑', '↓', '←', '→', '↵', '⌫', '…', '·', '›', '─', '⠋', '▶':
		return true
	}
	return false
}

// truncateRight shortens s to fit visual width w, appending "…" if cut.
func truncateRight(s string, w int) string {
	if w <= 0 {
		return ""
	}
	cur := 0
	out := make([]rune, 0, w)
	for _, r := range s {
		rw := 1
		if r > 0x7f {
			rw = 2
		}
		if cur+rw > w-1 {
			out = append(out, '…')
			return string(out)
		}
		out = append(out, r)
		cur += rw
	}
	return s
}

// padRight pads s with spaces to visual width w.
func padRight(s string, w int) string {
	cur := visualWidth(s)
	if cur >= w {
		return s
	}
	return s + strings.Repeat(" ", w-cur)
}

