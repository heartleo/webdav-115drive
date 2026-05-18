package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// maxContentWidth caps the rendered TUI width. On wider terminals the body
// is centered horizontally to keep lines short and readable.
const maxContentWidth = 120

var (
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(borderFg)
	errorStyle = lipgloss.NewStyle().Foreground(errFg)
)

// contentWidth returns the rendered width: terminal width clamped to maxContentWidth.
func contentWidth(termWidth int) int {
	if termWidth < maxContentWidth {
		return termWidth
	}
	return maxContentWidth
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.width < 30 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			"terminal too small (≥30 cols)",
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	cw := contentWidth(m.width)
	innerW := cw - 4 // borders (2) + padding (2)
	if innerW < 10 {
		innerW = 10
	}

	paneAnchor := m.statusIndicator() + "  ?"
	// Reserve space in top border: 4 corner chars + 2×"[ " + 2×" ]" = 12.
	titleBudget := cw - 12 - lipgloss.Width(paneAnchor)
	if titleBudget < 8 {
		titleBudget = 8
	}
	paneTitle := m.buildPaneTitle(titleBudget)

	paneInner := strings.Join([]string{
		"",
		columnHeader(innerW, m.sort),
		dividerLine(innerW),
		m.list.View(),
	}, "\n")

	body := renderPane(paneTitle, paneAnchor, paneInner, cw)

	switch {
	case m.showHelp:
		return overlay(m.helpView(), cw, m.height)
	case m.showInfo:
		if it, ok := m.currentItem(); ok {
			return overlay(m.infoView(it, cw), cw, m.height)
		}
	}
	return body
}

var (
	paneBorderStyle = lipgloss.NewStyle().Foreground(borderFg)
	paneTitleStyle  = lipgloss.NewStyle().Foreground(dimFg)
	paneFilterStyle = lipgloss.NewStyle().Foreground(accentFg).Bold(true)
)

// buildPaneTitle returns the title segment shown in the pane's border-top.
// Fuses the breadcrumb into the frame (no separate top line). The breadcrumb
// is collapsed to fit `budget` visual columns; entry count and the optional
// filter chip are appended on the right.
func (m model) buildPaneTitle(budget int) string {
	var suffix string
	if filter := m.list.FilterValue(); filter != "" {
		matched := len(m.list.VisibleItems())
		suffix = " " + paneFilterStyle.Render("filter:"+filter) +
			fmt.Sprintf(" (%d/%d)", matched, len(m.entries))
	}
	crumb := fitBreadcrumb(m.cwd, budget-lipgloss.Width(suffix))
	return crumb + suffix
}

// fitBreadcrumb renders cwd as a breadcrumb collapsed to fit width.
func fitBreadcrumb(cwd string, width int) string {
	if width < 1 {
		width = 1
	}
	segs := splitPath(cwd)
	left := buildBreadcrumb(segs)
	if visualWidth(left) <= width || len(segs) <= 3 {
		if visualWidth(left) > width {
			return truncateRight(left, width)
		}
		return left
	}
	for keepFront := len(segs) - 2; keepFront >= 1; keepFront-- {
		collapsed := append([]string{}, segs[:keepFront]...)
		collapsed = append(collapsed, "…")
		collapsed = append(collapsed, segs[len(segs)-2:]...)
		left = buildBreadcrumb(collapsed)
		if visualWidth(left) <= width {
			return left
		}
	}
	return truncateRight(left, width)
}

// renderPane wraps content in a rounded border with title (left) and anchor
// (right) embedded in the top border line.
func renderPane(title, anchor, content string, width int) string {
	left := "[ " + title + " ]"
	right := "[ " + anchor + " ]"
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	// fixed border chars: "╭─" (2) + "─╮" (2)
	fill := width - 4 - leftW - rightW
	if fill < 0 {
		fill = 0
	}
	topLine := paneBorderStyle.Render("╭─") +
		paneTitleStyle.Render("[ ") + title + paneTitleStyle.Render(" ]") +
		paneBorderStyle.Render(strings.Repeat("─", fill)) +
		paneTitleStyle.Render("[ ") + anchor + paneTitleStyle.Render(" ]") +
		paneBorderStyle.Render("─╮")
	botLine := paneBorderStyle.Render("╰" + strings.Repeat("─", width-2) + "╯")

	innerW := width - 4 // borders + padding
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = paneBorderStyle.Render("│") + " " +
			padRightStyled(line, innerW) + " " +
			paneBorderStyle.Render("│")
	}
	return topLine + "\n" + strings.Join(lines, "\n") + "\n" + botLine
}

// statusIndicator renders the status cell: error, loading spinner, or
// [idx/total] when idle.
func (m model) statusIndicator() string {
	idx := fmt.Sprintf("[%d/%d]", m.list.Index()+1, len(m.entries))
	switch {
	case m.err != nil:
		return errorStyle.Render("error: "+m.err.Error()) + " " + idx
	case m.loading:
		return m.spinner.View() + " " + idx
	}
	return idx
}

func (m model) helpView() string {
	return modalStyle.Render(m.help.View(m.keys))
}

func (m model) infoView(it item, cw int) string {
	type row struct{ label, value string }
	rows := []row{
		{"path", it.info.Path},
		{"name", it.info.Name},
	}
	if !it.info.IsDir {
		rows = append(rows, row{"size", fmt.Sprintf("%s B (%s)", bytesWithCommas(it.info.Size), humanSize(it.info.Size))})
	}
	rows = append(rows,
		row{"modified", it.info.ModTime.Format("2006-01-02 15:04")},
		row{"sha1", valueOrDash(it.info.ETag)},
	)
	if !it.info.IsDir {
		rows = append(rows, row{"pickcode", valueOrDash(it.info.PickCode)})
	}

	// Cap modal at content width. Inner = cw - border(2) - padding(4).
	modalW := cw - 2
	if modalW < 20 {
		modalW = 20
	}
	innerW := modalW - 6
	if innerW < 14 {
		innerW = 14
	}
	const labelW = 8
	valueW := innerW - labelW - 2
	if valueW < 8 {
		valueW = 8
	}

	var b strings.Builder
	for _, r := range rows {
		lines := hardWrap(r.value, valueW)
		for i, ln := range lines {
			if i == 0 {
				b.WriteString(padRight(r.label, labelW))
				b.WriteString("  ")
			} else {
				b.WriteString(strings.Repeat(" ", labelW+2))
			}
			b.WriteString(ln)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n[esc/i] close")

	title := it.info.Name
	if it.info.IsDir {
		title = dirIcon + " " + title
	}
	titleLines := hardWrap(title, innerW)

	return modalStyle.
		Width(innerW).
		BorderTop(true).
		Render(titleBar(strings.Join(titleLines, "\n")) + "\n" + b.String())
}

// hardWrap splits s into chunks of at most w runes. Unlike word-wrap it
// breaks anywhere, so opaque tokens (sha1, pickcode, long paths) still fit.
func hardWrap(s string, w int) []string {
	if w <= 0 {
		return []string{s}
	}
	rs := []rune(s)
	if len(rs) <= w {
		return []string{s}
	}
	var out []string
	for len(rs) > w {
		out = append(out, string(rs[:w]))
		rs = rs[w:]
	}
	if len(rs) > 0 {
		out = append(out, string(rs))
	}
	return out
}

// titleBar renders a centered title segment that visually separates it from
// the body. It does not draw a border itself; modalStyle adds the rounded
// border around the whole content.
func titleBar(title string) string {
	return lipgloss.NewStyle().Bold(true).Render(title)
}

func valueOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// overlay centers content on a w×h whitespace canvas. The body is hidden
// while a modal is shown (modal semantics).
func overlay(content string, w, h int) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceChars(" "))
}

var (
	columnHeaderStyle = lipgloss.NewStyle().Foreground(dimFg)
	dividerStyle      = lipgloss.NewStyle().Foreground(dimFg)
	hintStyle         = lipgloss.NewStyle().Foreground(dimFg).Padding(0, 1)
)

func columnHeader(width int, sort sortMode) string {
	if width < 30 {
		return ""
	}
	icon := "  " // blank placeholder cell for the icon column (2 cells)
	const gap = "   " // matches sizeMtimeGap in delegate.go
	const cursorColW = 2 // matches delegate.go cursor prefix width

	nameLbl := styleSortLabel("Name", sort, sortNameAsc, sortNameDesc)
	sizeLbl := styleSortLabel("Size", sort, sortSizeAsc, sortSizeDesc)
	mtimeLbl := styleSortLabel("Modified", sort, sortMtimeAsc, sortMtimeDesc)

	avail := width - cursorColW - visualWidth(icon) - 1 - rowSizeColW - len(gap) - rowMtimeColW - 1
	if avail < 4 {
		avail = 4
	}
	name := padRightStyled(nameLbl, avail)
	size := padRightStyled(sizeLbl, rowSizeColW)
	mtime := padRightStyled(mtimeLbl, rowMtimeColW)
	return columnHeaderStyle.Render(fmt.Sprintf("  %s %s %s%s%s", icon, name, size, gap, mtime))
}

var activeSortStyle = lipgloss.NewStyle().Foreground(accentFg).Bold(true)

// styleSortLabel renders a column label with sort arrow. Active column is
// rendered with accent style (magenta + bold).
func styleSortLabel(label string, sort, asc, desc sortMode) string {
	arrow := sortArrow(sort, asc, desc)
	full := label + arrow
	if arrow != "" {
		return activeSortStyle.Render(full)
	}
	return full
}

// padRightStyled pads a (possibly ANSI-styled) string to visual width w by
// appending spaces. Uses lipgloss.Width to ignore escape sequences.
func padRightStyled(s string, w int) string {
	cur := lipgloss.Width(s)
	if cur >= w {
		return s
	}
	return s + strings.Repeat(" ", w-cur)
}

// sortArrow returns "↑" if sort==asc, "↓" if sort==desc, else "".
func sortArrow(sort, asc, desc sortMode) string {
	switch sort {
	case asc:
		return "↑"
	case desc:
		return "↓"
	}
	return ""
}

func dividerLine(width int) string {
	return dividerStyle.Render(strings.Repeat("─", width))
}

