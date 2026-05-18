package tui

import "strings"

const breadcrumbSep = " › "

func splitPath(p string) []string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func buildBreadcrumb(segs []string) string {
	if len(segs) == 0 {
		return "/"
	}
	return "/" + strings.Join(segs, breadcrumbSep)
}
