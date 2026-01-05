package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type Drive struct {
	conf         *DriveConfig
	client       *driver.Pan115Client
	reverseProxy *httputil.ReverseProxy
	limiter      *rate.Limiter
	cache        *cache.Cache
}

type jarTransport struct {
	tripper http.RoundTripper
	jar     http.CookieJar
}

func (t *jarTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for _, v := range t.jar.Cookies(req.URL) {
		req.AddCookie(v)
	}
	return t.tripper.RoundTrip(req)
}

func newDrive(conf *DriveConfig) (*Drive, error) {
	credential := &driver.Credential{
		UID:  conf.UID,
		CID:  conf.CID,
		SEID: conf.SEID,
		KID:  conf.KID,
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	restyClient := resty.New().SetCookieJar(jar)

	client := driver.New(driver.WithRestyClient(restyClient)).
		SetUserAgent(driver.UA115Browser).ImportCredential(credential)

	if err := client.LoginCheck(); err != nil {
		return nil, fmt.Errorf("drive login failed: %w", err)
	}

	reverseProxy := &httputil.ReverseProxy{
		Transport: &jarTransport{
			tripper: restyClient.GetClient().Transport,
			jar:     jar,
		},
		Director: func(req *http.Request) {
			req.Header.Set("Referer", driver.CookieUrl)
			req.Header.Set("User-Agent", driver.UA115Browser)
			req.Header.Set("Host", req.Host)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("reverse proxy failed", slog.Any("error", err), slog.String("url", r.URL.String()))
		},
		ModifyResponse: func(response *http.Response) error {
			if response.StatusCode >= http.StatusBadRequest {
				b, _ := io.ReadAll(response.Body)
				slog.Warn("reverse proxy failed", slog.Any("status", response.Status),
					slog.Any("body", string(b)))
			}
			return nil
		},
	}

	expire := time.Duration(conf.CacheExpire) * time.Minute

	fs := &Drive{
		conf:         conf,
		client:       client,
		reverseProxy: reverseProxy,
		limiter:      rate.NewLimiter(rate.Every(time.Second), conf.Rate),
		cache:        cache.New(expire, expire*2),
	}

	return fs, nil
}

func (d *Drive) Stat(ctx context.Context, p string) (*Info, error) {
	p = path.Join("/", p)
	if p == "/" {
		return &Info{
			Path:    "/",
			Name:    "/",
			IsDir:   true,
			ModTime: time.Now(),
		}, nil
	}

	dir, name := path.Split(strings.TrimSuffix(p, "/"))

	files, err := d.ReadDir(ctx, dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.Name == name {
			return f, nil
		}
	}

	return nil, errors.New("file not found")
}

func (d *Drive) ReadDir(ctx context.Context, p string) ([]*Info, error) {
	p = path.Join("/", p)

	result, err := d.fetchCache(ctx, d.cacheKeyDir(p), func() (any, error) {
		dirID := "0"

		if dirResp, err := d.client.DirName2CID(p); err == nil {
			dirID = string(dirResp.CategoryID)
		}

		var files *[]driver.File

		err := d.checkRateLimit(ctx, func() error {
			var e error
			files, e = d.client.List(dirID)
			return e
		})
		if err != nil {
			return nil, fmt.Errorf("list files failed: %w", err)
		}

		infos := make([]*Info, 0, len(*files))

		for _, f := range *files {
			infos = append(infos, &Info{
				Path:     path.Join(p, f.Name),
				Name:     f.Name,
				IsDir:    f.IsDirectory,
				Size:     f.Size,
				ModTime:  f.UpdateTime,
				ETag:     f.Sha1,
				PickCode: f.PickCode,
			})
		}

		return infos, nil
	})
	if err != nil {
		return nil, err
	}

	return result.([]*Info), nil
}

func (d *Drive) Open(ctx context.Context, p string) (io.ReadSeeker, *Info, error) {
	info, err := d.Stat(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	return nil, info, nil
}

func (d *Drive) ServeContent(w http.ResponseWriter, r *http.Request, info *Info) error {
	if info.PickCode == "" {
		return errors.New("pick code not found")
	}

	result, err := d.fetchCache(r.Context(), d.cacheKeyDownload(info.PickCode), func() (any, error) {
		downloadInfo, err := d.client.Download(info.PickCode)
		if err != nil {
			return nil, fmt.Errorf("download failed: %w", err)
		}
		return downloadInfo.Url.Url, nil
	})
	if err != nil {
		return err
	}

	du, err := url.Parse(result.(string))
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}

	slog.Debug("serve content",
		slog.String("path", info.Path),
		slog.String("name", info.Name),
		slog.String("pickCode", info.PickCode),
		slog.String("range", r.Header.Get("Range")),
		slog.String("url", du.String()),
	)

	r.URL = du
	r.Host = du.Host
	d.reverseProxy.ServeHTTP(w, r)

	return nil
}
