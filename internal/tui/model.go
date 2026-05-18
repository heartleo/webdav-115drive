package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/heartleo/webdav-115drive/internal/drive"
)

// dirLoadedMsg is emitted when a ReadDir call returns successfully.
type dirLoadedMsg struct {
	path    string
	entries []*drive.Info
}

// errMsg is emitted when a ReadDir call fails. The path lets the model
// discard responses that no longer match the current cwd.
type errMsg struct {
	path string
	err  error
}

// item is the list.Item adapter for *drive.Info.
type item struct{ info *drive.Info }

func (i item) FilterValue() string { return i.info.Name }

// cancelCell holds the cancel func for the current in-flight load. It is a
// pointer so mutations are visible across model value copies (bubbletea
// passes model by value through Update).
type cancelCell struct{ fn context.CancelFunc }

type model struct {
	ctx  context.Context
	fs   drive.FileSystem
	keys keyMap

	cwd     string
	entries []*drive.Info

	list    list.Model
	spinner spinner.Model
	help    help.Model

	loading  bool
	cancel   *cancelCell
	err      error
	showInfo bool
	showHelp bool

	sort sortMode

	width  int
	height int
}

func newModel(ctx context.Context, fs drive.FileSystem) model {
	l := list.New(nil, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	hp := help.New()
	hp.ShowAll = true
	hp.Styles.FullKey = lipgloss.NewStyle().Foreground(dimFg)
	hp.Styles.FullDesc = lipgloss.NewStyle().Foreground(dimFg)
	hp.Styles.FullSeparator = lipgloss.NewStyle().Foreground(dimFg)
	hp.Styles.ShortKey = lipgloss.NewStyle().Foreground(dimFg)
	hp.Styles.ShortDesc = lipgloss.NewStyle().Foreground(dimFg)
	hp.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(dimFg)

	return model{
		ctx:     ctx,
		fs:      fs,
		keys:    defaultKeys(),
		cwd:     "/",
		list:    l,
		spinner: sp,
		help:    hp,
		cancel:  &cancelCell{},
		loading: true,
		sort:    sortNameAsc,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.startLoad(m.cwd), m.spinner.Tick)
}

// startLoad cancels any in-flight load and returns a tea.Cmd that fetches p
// with a fresh cancellable context. Cancel func stashed on shared cancelCell
// so the next navigation cancels this in-flight call.
func (m model) startLoad(p string) tea.Cmd {
	if m.cancel.fn != nil {
		m.cancel.fn()
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.cancel.fn = cancel

	fs := m.fs
	mode := m.sort
	return func() tea.Msg {
		entries, err := fs.ReadDir(ctx, p)
		if err != nil {
			if ctx.Err() != nil {
				return nil // canceled
			}
			return errMsg{path: p, err: err}
		}
		return dirLoadedMsg{path: p, entries: applySort(entries, mode)}
	}
}
