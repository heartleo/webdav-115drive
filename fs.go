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

	"github.com/SheltonZhu/115driver/pkg/driver"
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
	client       *driver.Pan115Client
	reverseProxy *httputil.ReverseProxy
	limiter      *rate.Limiter
	cache        *cache.Cache
	mu           sync.RWMutex
}

const UA = "Mozilla/5.0 115Browser/23.9.3.2"

func NewDrive115FS(conf Drive115Config) (*Drive115FS, error) {

	cr := &driver.Credential{
		UID:  conf.UID,
		CID:  conf.CID,
		SEID: conf.SEID,
		KID:  conf.KID,
	}

	client := driver.Default().SetUserAgent(UA).ImportCredential(cr)
	if err := client.LoginCheck(); err != nil {
		return nil, fmt.Errorf("create 115 drive failed: %w", err)
	}

	reverseProxy := &httputil.ReverseProxy{
		Transport: client.Client.GetClient().Transport,
		Director: func(req *http.Request) {
			req.Header.Set("Referer", "https://115.com/")
			req.Header.Set("User-Agent", UA)
			req.Header.Set("Host", req.Host)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("reverse proxy failed", slog.Any("error", err), slog.String("url", r.URL.String()))
		},
		ModifyResponse: func(response *http.Response) error {
			if response.StatusCode >= http.StatusBadRequest {
				b, _ := io.ReadAll(response.Body)
				slog.Warn("reverse proxy failed", slog.Any("status", response.Status),
					slog.Any("message", string(b)))
			}
			return nil
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

	var dirID string

	dirResp, err := d.client.DirName2CID(p)
	if err == nil {
		dirID = string(dirResp.CategoryID)
	}

	if dirID == "" {
		dirID = "0"
	}

	if err := d.waitLimit(ctx); err != nil {
		return nil, err
	}

	files, err := d.client.List(dirID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	infos := make([]Info, 0, len(*files))
	for _, f := range *files {
		infos = append(infos, Info{
			Path:     path.Join(p, f.Name),
			Name:     f.Name,
			IsDir:    f.IsDirectory,
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

		downloadInfo, err := d.client.Download(info.PickCode)
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

	if err := d.waitLimit(ctx); err != nil {
		return err
	}

	slog.Debug("serving content",
		slog.String("name", info.Name),
		slog.String("pickCode", info.PickCode),
		slog.String("range", r.Header.Get("Range")),
		slog.String("url", du.String()),
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
