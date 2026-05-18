package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	PgUp   key.Binding
	PgDown key.Binding
	Top    key.Binding
	Bottom key.Binding
	Enter  key.Binding
	Back   key.Binding
	Filter key.Binding
	Sort   key.Binding
	Info   key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		PgUp:   key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		PgDown: key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		Top:    key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom: key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Enter:  key.NewBinding(key.WithKeys("enter", "l", "right"), key.WithHelp("↵/l", "enter dir")),
		Back:   key.NewBinding(key.WithKeys("backspace", "h", "left"), key.WithHelp("⌫/h", "parent")),
		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Sort:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Info:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "info")),
		Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp / FullHelp implement bubbles/help.KeyMap so the help overlay
// is auto-rendered from the keymap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Filter, k.Sort, k.Info, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PgUp, k.PgDown, k.Top, k.Bottom},
		{k.Enter, k.Back, k.Filter, k.Sort},
		{k.Info, k.Help, k.Quit},
	}
}
