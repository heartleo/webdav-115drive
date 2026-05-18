package tui

import (
	"path/filepath"
	"strings"

	"github.com/heartleo/webdav-115drive/internal/drive"
)

// extByIcon defines the (icon, extensions) buckets in source order. The
// init() flattens this into extToIcon for O(1) lookup.
var extByIcon = []struct {
	icon string
	exts []string
}{
	{"🎬", []string{"mp4", "mkv", "avi", "mov", "wmv", "flv", "webm", "m4v", "ts", "mpg", "mpeg", "rmvb"}},
	{"🖼", []string{"jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "heic", "tiff", "ico"}},
	{"🎵", []string{"mp3", "flac", "wav", "aac", "m4a", "ogg", "opus", "wma", "ape"}},
	{"📦", []string{"zip", "rar", "7z", "tar", "gz", "bz2", "xz", "iso"}},
	{"📖", []string{"epub", "mobi", "azw", "azw3", "azw4", "kfx", "prc", "pdb", "kpf", "fb2", "lit"}},
	{"💬", []string{"srt", "ass", "ssa", "vtt", "sub", "idx", "sup", "smi", "sbv", "lrc"}},
	{"📄", []string{"txt", "md", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "csv", "json", "xml", "log"}},
	{"⚙", []string{"exe", "dmg", "app", "apk", "msi", "deb", "rpm", "dll", "so"}},
}

var extToIcon map[string]string

func init() {
	extToIcon = make(map[string]string, 64)
	for _, e := range extByIcon {
		for _, x := range e.exts {
			extToIcon[x] = e.icon
		}
	}
}

const (
	dirIcon     = "📁"
	unknownIcon = "❔"
)

// iconFor returns the emoji for an entry. Directories always get dirIcon.
// Files use the extension map; unknown extensions fall back to unknownIcon.
func iconFor(info *drive.Info) string {
	if info.IsDir {
		return dirIcon
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(info.Name)), ".")
	if ext == "" {
		return unknownIcon
	}
	if icon, ok := extToIcon[ext]; ok {
		return icon
	}
	return unknownIcon
}
