package handler

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/heartleo/webdav-115drive/internal/drive"
	"github.com/heartleo/webdav-115drive/internal/webdav"
)

type Handler struct {
	FS       drive.FileSystem
	BasePath string
}

func New(fs drive.FileSystem, basePath string) (*Handler, error) {
	if fs == nil {
		return nil, errors.New("fs is required")
	}
	return &Handler{
		FS:       fs,
		BasePath: strings.TrimRight(basePath, "/"),
	}, nil
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
		slog.Error("method not allowed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}
}

func (h *Handler) handleOptions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("DAV", "1")
	w.Header().Set("MS-Author-Via", "DAV")
	w.Header().Set("Allow", "OPTIONS, GET, HEAD, PROPFIND")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetHead(w http.ResponseWriter, r *http.Request) {
	p, ok := h.cleanPath(r.URL.Path)
	if !ok {
		slog.Warn("bad path", slog.String("path", r.URL.Path))
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	info, ok := h.statOrError(w, r, p)
	if !ok {
		return
	}

	if info.IsDir {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		slog.Error("bad method", slog.String("path", p))
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

	cs, ok := h.FS.(drive.ContentServer)
	if !ok {
		slog.Error("file not readable", slog.String("path", p))
		http.Error(w, "file not readable", http.StatusInternalServerError)
		return
	}

	if err := cs.ServeContent(w, r, info); err != nil {
		slog.Error("serve content failed", slog.String("path", p), slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) handlePropfind(w http.ResponseWriter, r *http.Request) {
	p, ok := h.cleanPath(r.URL.Path)
	if !ok {
		slog.Warn("bad path", slog.String("path", r.URL.Path))
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	depth := r.Header.Get("Depth")
	if depth == "" {
		depth = "1"
	}

	if depth != "0" && depth != "1" {
		slog.Error("bad depth",
			slog.String("path", r.URL.Path),
			slog.String("depth", depth),
		)
		http.Error(w, "bad depth", http.StatusForbidden)
		return
	}

	root, ok := h.statOrError(w, r, p)
	if !ok {
		return
	}

	var children []*drive.Info

	if root.IsDir && depth == "1" {
		var err error
		children, err = h.FS.ReadDir(r.Context(), p)
		if err != nil {
			slog.Error("read dir failed", slog.String("path", p), slog.Any("error", err))
			http.Error(w, "read dir failed", http.StatusInternalServerError)
			return
		}
	}

	responses := []webdav.DavResponse{h.makeResponse(root)}
	for _, v := range children {
		responses = append(responses, h.makeResponse(v))
	}

	ms := webdav.MultiStatus{
		XmlnsD:   "DAV:",
		Response: responses,
	}

	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(webdav.XmlHeader))
	_ = webdav.XMLEncoder(w).Encode(ms)
}

func (h *Handler) makeResponse(info *drive.Info) webdav.DavResponse {
	href := h.toHref(info.Path, info.IsDir)
	etag := h.ensureETag(info)

	props := webdav.Prop{
		DisplayName: info.Name,
		GetETag:     etag,
		LastMod:     info.ModTime.UTC().Format(http.TimeFormat),
	}

	if info.IsDir {
		props.ResourceType = &webdav.ResourceType{Collection: &struct{}{}}
	} else {
		props.ContentLength = fmt.Sprintf("%d", info.Size)
		props.ResourceType = &webdav.ResourceType{}
	}

	return webdav.DavResponse{
		Href: href,
		Propstat: webdav.PropStat{
			Prop:   props,
			Status: "HTTP/1.1 200 OK",
		},
	}
}

func (h *Handler) statOrError(w http.ResponseWriter, r *http.Request, p string) (*drive.Info, bool) {
	info, err := h.FS.Stat(r.Context(), p)
	if err != nil {
		if errors.Is(err, drive.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			slog.Error("stat failed", slog.String("path", p), slog.Any("error", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return nil, false
	}
	return info, true
}

func (h *Handler) ensureETag(info *drive.Info) string {
	if info.ETag != "" {
		return quoteETag(info.ETag)
	}
	sum := sha1.Sum(fmt.Appendf(nil, "%d:%d", info.Size, info.ModTime.UnixNano()))
	return quoteETag(hex.EncodeToString(sum[:]))
}

func quoteETag(s string) string {
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s
	}
	return `"` + s + `"`
}

func (h *Handler) cleanPath(urlPath string) (string, bool) {
	if h.BasePath != "" {
		if !strings.HasPrefix(urlPath, h.BasePath) {
			return "", false
		}
		urlPath = strings.TrimPrefix(urlPath, h.BasePath)
	}
	return path.Join("/", urlPath), true
}

func (h *Handler) toHref(p string, isDir bool) string {
	joined := path.Join("/", h.BasePath, p)
	segments := strings.Split(strings.TrimPrefix(joined, "/"), "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	href := "/" + strings.Join(segments, "/")
	if isDir && href != "/" {
		href += "/"
	}
	return href
}
