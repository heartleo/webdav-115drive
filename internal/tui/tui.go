// Package tui implements an interactive terminal browser for a 115 Drive
// account. It is read-only and consumes drive.FileSystem.
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/heartleo/webdav-115drive/internal/drive"
)

// Run launches the TUI and blocks until the user quits or ctx is canceled.
func Run(ctx context.Context, fs drive.FileSystem) error {
	m := newModel(ctx, fs)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
