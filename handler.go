package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

type Handler struct {
	FS       FS
	BasePath string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodOptions:
		h.handleOptions(w, r)
	case http.MethodGet, http.MethodHead:
		h.handleGetHead(w, r)
	case "PROPFIND":
		h.handlePropfind(w, r)
	default:
		w.Header().Set("Allow", "OPTIONS, GET, HEAD, PROPFIND")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleOptions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("DAV", "1")
	w.Header().Set("MS-Author-Via", "DAV")
	w.Header().Set("Allow", "OPTIONS, GET, HEAD, PROPFIND")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetHead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := h.cleanPath(r.URL.Path)
	if !ok {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	info, err := h.FS.Stat(ctx, p)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if info.IsDir {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}

	_, _, err = h.FS.Open(ctx, p)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	etag := h.ensureETag(info)
	lastMod := info.ModTime.UTC().Format(http.TimeFormat)

	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", lastMod)
	w.Header().Set("Accept-Ranges", "bytes")

	if s := r.Header.Get("If-None-Match"); s != "" && strings.Contains(s, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if s := r.Header.Get("If-Modified-Since"); s != "" {
		if t, e := time.Parse(http.TimeFormat, s); e == nil && !info.ModTime.After(t) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	contentType := mime.TypeByExtension(path.Ext(info.Name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	if d115fs, ok := h.FS.(*Drive115); ok {
		if err := d115fs.ServeContent(w, r, info); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, "file not readable", http.StatusInternalServerError)
	return
}

func (h *Handler) handlePropfind(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, ok := h.cleanPath(r.URL.Path)
	if !ok {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	depth := r.Header.Get("Depth")
	if depth == "" {
		depth = "1"
	}
	if depth != "0" && depth != "1" {
		http.Error(w, "Depth not supported", http.StatusForbidden)
		return
	}

	root, err := h.FS.Stat(ctx, p)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var children []Info
	if root.IsDir && depth == "1" {
		children, err = h.FS.ReadDir(ctx, p)
		if err != nil {
			http.Error(w, "failed to read directory", http.StatusInternalServerError)
			return
		}
	}

	responses := []davResponse{h.makeResponse(r, root)}
	for _, c := range children {
		responses = append(responses, h.makeResponse(r, c))
	}

	ms := multistatus{
		XmlnsD:   "DAV:",
		Response: responses,
	}

	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(xmlHeader))
	_ = xmlEncoder(w).Encode(ms)
}

func (h *Handler) makeResponse(r *http.Request, info Info) davResponse {
	href := h.toHref(r, info.Path, info.IsDir)
	etag := h.ensureETag(info)

	props := prop{
		DisplayName: info.Name,
		GetETag:     etag,
		LastMod:     info.ModTime.UTC().Format(http.TimeFormat),
	}
	if info.IsDir {
		props.ResourceType = &resourcetype{Collection: &struct{}{}}
	} else {
		props.ContentLength = fmt.Sprintf("%d", info.Size)
		props.ResourceType = &resourcetype{}
	}
	return davResponse{
		Href: href,
		Propstat: propstat{
			Prop:   props,
			Status: "HTTP/1.1 200 OK",
		},
	}
}

func (h *Handler) ensureETag(info Info) string {
	if info.ETag != "" {
		return quoteETag(info.ETag)
	}
	sum := sha1.Sum([]byte(fmt.Sprintf("%d:%d", info.Size, info.ModTime.UnixNano())))
	return quoteETag(hex.EncodeToString(sum[:]))
}

func quoteETag(s string) string {
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s
	}
	return `"` + s + `"`
}

func (h *Handler) cleanPath(urlPath string) (string, bool) {
	p := urlPath

	if h.BasePath != "" {
		if !strings.HasPrefix(p, h.BasePath) {
			return "", false
		}
		p = strings.TrimPrefix(p, h.BasePath)
	}

	if p == "" {
		p = "/"
	}

	cp := path.Clean("/" + p)

	if strings.Contains(cp, "..") {
		return "", false
	}

	return cp, true
}

func (h *Handler) toHref(r *http.Request, p string, isDir bool) string {
	href := p

	if h.BasePath != "" {
		href = path.Join(h.BasePath, p)
		if !strings.HasPrefix(href, "/") {
			href = "/" + href
		}
	}

	if isDir && !strings.HasSuffix(href, "/") {
		href += "/"
	}

	return href
}
