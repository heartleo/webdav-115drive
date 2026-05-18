package tui

import (
	"path"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	breadcrumbHeight   = 1
	paneTopHeight      = 1
	columnHeaderHeight = 1
	dividerHeight      = 1
	paneBotHeight      = 1

	chromeHeight = breadcrumbHeight + paneTopHeight + columnHeaderHeight + dividerHeight + paneBotHeight
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(contentWidth(msg.Width)-4, msg.Height-chromeHeight)

	case dirLoadedMsg:
		if msg.path != m.cwd {
			return m, nil // stale
		}
		m.loading = false
		m.err = nil
		m.entries = msg.entries
		items := make([]list.Item, len(msg.entries))
		for i, e := range msg.entries {
			items[i] = item{info: e}
		}
		m.list.SetItems(items)
		m.list.Select(0)

	case errMsg:
		if msg.path != m.cwd {
			return m, nil // stale
		}
		m.loading = false
		m.err = msg.err

	case tea.KeyMsg:
		// Modal dismissal first.
		if m.showInfo || m.showHelp {
			if msg.String() == "esc" || msg.String() == "i" || msg.String() == "?" || key.Matches(msg, m.keys.Quit) {
				m.showInfo = false
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		// When the embedded list is in filter-input mode, forward keys to it
		// so typing the filter and Esc/Enter behave correctly.
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = true
			return m, nil
		case key.Matches(msg, m.keys.Info):
			if _, ok := m.currentItem(); ok {
				m.showInfo = true
			}
			return m, nil
		case key.Matches(msg, m.keys.Filter):
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		case key.Matches(msg, m.keys.Sort):
			m.sort = (m.sort + 1) % sortModeCount
			applySort(m.entries, m.sort)
			items := make([]list.Item, len(m.entries))
			for i, e := range m.entries {
				items[i] = item{info: e}
			}
			m.list.SetItems(items)
			m.list.Select(0)
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if m.loading {
				return m, nil
			}
			if it, ok := m.currentItem(); ok && it.info.IsDir {
				m.cwd = path.Join(m.cwd, it.info.Name)
				m.loading = true
				return m, m.startLoad(m.cwd)
			}
			return m, nil
		case key.Matches(msg, m.keys.Back):
			if m.loading || m.cwd == "/" {
				return m, nil
			}
			m.cwd = path.Dir(m.cwd)
			if m.cwd == "" {
				m.cwd = "/"
			}
			m.loading = true
			return m, m.startLoad(m.cwd)
		}
	}

	// Default: pass to the list (handles up/down/pgup/pgdown/home/end).
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	if m.loading {
		var scmd tea.Cmd
		m.spinner, scmd = m.spinner.Update(msg)
		cmds = append(cmds, scmd)
	}

	return m, tea.Batch(cmds...)
}

// currentItem returns the entry under the cursor, if any.
func (m model) currentItem() (item, bool) {
	sel := m.list.SelectedItem()
	if sel == nil {
		return item{}, false
	}
	it, ok := sel.(item)
	return it, ok
}
