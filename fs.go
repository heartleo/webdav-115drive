package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/heartleo/webdav-115drive/drive"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type FS interface {
	Stat(ctx context.Context, p string) (Info, error)
	ReadDir(ctx context.Context, p string) ([]Info, error)
	Open(ctx context.Context, p string) (io.ReadSeeker, Info, error)
}

type Info struct {
	Path     string
	Name     string
	IsDir    bool
	Size     int64
	ModTime  time.Time
	ETag     string
	PickCode string
}

type Drive115FS struct {
	client       *drive.Client
	reverseProxy *httputil.ReverseProxy
	limiter      *rate.Limiter
	cache        *cache.Cache
	mu           sync.RWMutex
}

func NewDrive115FS(conf Drive115Config) (*Drive115FS, error) {
	client, err := drive.New(conf.UID, conf.CID, conf.SEID, conf.KID)
	if err != nil {
		return nil, fmt.Errorf("create 115 drive failed: %w", err)
	}

	reverseProxy := &httputil.ReverseProxy{
		Transport: client.HttpClient().Transport,
		Director: func(req *http.Request) {
			req.Header.Set("Referer", drive.Referer)
			req.Header.Set(drive.UAKey, drive.UA115Browser)
			req.Header.Set("Host", req.Host)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("reverse proxy failed", slog.Any("error", err), slog.String("url", r.URL.String()))
		},
	}

	fs := &Drive115FS{
		client:       client,
		reverseProxy: reverseProxy,
		limiter:      rate.NewLimiter(rate.Every(time.Second), conf.Rate),
		cache:        cache.New(1*time.Minute, 5*time.Minute),
	}

	return fs, nil
}

func (d *Drive115FS) waitLimit(ctx context.Context) error {
	if d.limiter == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := d.limiter.Wait(ctx); err != nil {
		return errors.New("rate limit exceeded")
	}
	return nil
}

func (d *Drive115FS) Stat(ctx context.Context, p string) (Info, error) {
	p = cleanPath(p)
	if p == "/" {
		return Info{
			Path:    "/",
			Name:    "/",
			IsDir:   true,
			ModTime: time.Now(),
		}, nil
	}

	dir, name := path.Split(strings.TrimSuffix(p, "/"))

	files, err := d.ReadDir(ctx, dir)
	if err != nil {
		return Info{}, err
	}

	for _, f := range files {
		if f.Name == name {
			return f, nil
		}
	}

	return Info{}, errors.New("file not found")
}

func (d *Drive115FS) ReadDir(ctx context.Context, p string) ([]Info, error) {
	p = cleanPath(p)
	cacheKey := "dir:" + p

	d.mu.RLock()
	if cached, ok := d.cache.Get(cacheKey); ok {
		d.mu.RUnlock()
		return cached.([]Info), nil
	}
	d.mu.RUnlock()

	if err := d.waitLimit(ctx); err != nil {
		return nil, err
	}

	dirID, err := d.client.DirID(p)
	if err != nil {
		return nil, fmt.Errorf("failed to get dir ID: %w", err)
	}

	if err := d.waitLimit(ctx); err != nil {
		return nil, err
	}

	files, err := d.client.FileList(dirID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	infos := make([]Info, 0, len(files))
	for _, f := range files {
		infos = append(infos, Info{
			Path:     path.Join(p, f.Name),
			Name:     f.Name,
			IsDir:    f.IsDir,
			Size:     f.Size,
			ModTime:  f.UpdateTime,
			ETag:     f.Sha1,
			PickCode: f.PickCode,
		})
	}

	d.mu.Lock()
	d.cache.Set(cacheKey, infos, cache.DefaultExpiration)
	d.mu.Unlock()

	return infos, nil
}

func (d *Drive115FS) Open(ctx context.Context, p string) (io.ReadSeeker, Info, error) {
	info, err := d.Stat(ctx, p)
	if err != nil {
		return nil, Info{}, err
	}
	return nil, info, nil
}

func (d *Drive115FS) ServeContent(w http.ResponseWriter, r *http.Request, info Info) error {
	ctx := r.Context()

	if info.PickCode == "" {
		return errors.New("pick code not found")
	}

	cacheKey := "download:" + info.PickCode

	var downloadURL string
	if cached, ok := d.cache.Get(cacheKey); ok {
		downloadURL = cached.(string)
	} else {
		if err := d.waitLimit(ctx); err != nil {
			return err
		}

		downloadInfo, err := d.client.DownloadInfo(info.PickCode)
		if err != nil {
			return fmt.Errorf("failed to get download info: %w", err)
		}

		downloadURL = downloadInfo.Url.Url
		d.cache.Set(cacheKey, downloadURL, cache.DefaultExpiration)
	}

	du, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}

	r.URL = du
	r.Host = du.Host

	slog.Debug("serving content",
		slog.String("name", info.Name),
		slog.String("pickCode", info.PickCode),
		slog.String("range", r.Header.Get("Range")),
	)

	d.reverseProxy.ServeHTTP(w, r)

	return nil
}

func cleanPath(p string) string {
	if p == "" || p[0] != '/' {
		p = "/" + p
	}
	return path.Clean(p)
}
