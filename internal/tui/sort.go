package tui

import (
	"sort"
	"strings"

	"github.com/heartleo/webdav-115drive/internal/drive"
)

type sortMode int

const (
	sortNameAsc sortMode = iota
	sortNameDesc
	sortSizeAsc
	sortSizeDesc
	sortMtimeAsc
	sortMtimeDesc
	sortModeCount = iota
)

// applySort sorts entries in place: directories always come first; within each
// group, ordering is determined by mode. Returns the same slice for chaining.
func applySort(entries []*drive.Info, mode sortMode) []*drive.Info {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		switch mode {
		case sortNameAsc:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case sortNameDesc:
			return strings.ToLower(a.Name) > strings.ToLower(b.Name)
		case sortSizeAsc:
			if a.IsDir { // dirs: alphabetical within group, size unknown
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
			return a.Size < b.Size
		case sortSizeDesc:
			if a.IsDir {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
			return a.Size > b.Size
		case sortMtimeAsc:
			return a.ModTime.Before(b.ModTime)
		case sortMtimeDesc:
			return a.ModTime.After(b.ModTime)
		}
		return false
	})
	return entries
}
