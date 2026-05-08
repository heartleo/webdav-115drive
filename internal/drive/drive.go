package drive

import (
	"bytes"
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
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
)

type Options struct {
	UID         string
	CID         string
	SEID        string
	KID         string
	Rate        int
	CacheExpire int
}

func (o Options) validate() error {
	missing := make([]string, 0, 4)
	if o.UID == "" {
		missing = append(missing, "UID")
	}
	if o.CID == "" {
		missing = append(missing, "CID")
	}
	if o.SEID == "" {
		missing = append(missing, "SEID")
	}
	if o.KID == "" {
		missing = append(missing, "KID")
	}
	if len(missing) > 0 {
		return fmt.Errorf("drive credentials missing: %s", strings.Join(missing, ", "))
	}
	if o.Rate <= 0 {
		return fmt.Errorf("drive rate must be positive, got %d", o.Rate)
	}
	if o.CacheExpire <= 0 {
		return fmt.Errorf("drive cache_expire must be positive, got %d", o.CacheExpire)
	}
	return nil
}

type contextKey int

const pickCodeCtxKey contextKey = iota

type Drive struct {
	client       *driver.Pan115Client
	reverseProxy *httputil.ReverseProxy
	limiter      *rate.Limiter
	cache        *cache.Cache
	group        singleflight.Group
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

func New(opts Options) (*Drive, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	credential := &driver.Credential{
		UID:  opts.UID,
		CID:  opts.CID,
		SEID: opts.SEID,
		KID:  opts.KID,
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	restyClient := resty.New().SetCookieJar(jar)

	client := driver.New(driver.WithRestyClient(restyClient)).
		SetUserAgent(driver.UA115Browser).ImportCredential(credential)

	if err := client.LoginCheck(); err != nil {
		return nil, fmt.Errorf("drive login: %w", err)
	}

	expire := time.Duration(opts.CacheExpire) * time.Minute
	c := cache.New(expire, expire*2)

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
			slog.Warn("reverse proxy error",
				slog.Any("error", err),
				slog.String("url", r.URL.String()),
			)
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode >= http.StatusBadRequest {
				b, _ := io.ReadAll(resp.Body)
				resp.Body = io.NopCloser(bytes.NewReader(b))
				slog.Warn("reverse proxy upstream error",
					slog.String("status", resp.Status),
					slog.String("body", string(b)),
				)
				if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
					if pc, ok := resp.Request.Context().Value(pickCodeCtxKey).(string); ok && pc != "" {
						c.Delete("download:" + pc)
						slog.Debug("evicted stale download URL on 416", slog.String("pickCode", pc))
					}
				}
			}
			return nil
		},
	}

	return &Drive{
		client:       client,
		reverseProxy: reverseProxy,
		limiter:      rate.NewLimiter(rate.Every(time.Second), opts.Rate),
		cache:        c,
	}, nil
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

	return nil, ErrNotFound
}

func (d *Drive) ReadDir(ctx context.Context, p string) ([]*Info, error) {
	p = path.Join("/", p)

	result, err := d.fetchCache(d.cacheKeyDir(p), func() (any, error) {
		// root always maps to dirID "0"; DirName2CID is skipped to avoid an
		// unnecessary API call and because it may not handle "/" correctly.
		dirID := "0"
		if p != "/" {
			if err := d.checkRateLimit(ctx, func() error {
				dirResp, err := d.client.DirName2CID(p)
				if err != nil {
					return fmt.Errorf("resolve path %q: %w", p, err)
				}
				dirID = string(dirResp.CategoryID)
				return nil
			}); err != nil {
				return nil, err
			}
		}

		var files *[]driver.File

		if err := d.checkRateLimit(ctx, func() error {
			var e error
			files, e = d.client.List(dirID)
			return e
		}); err != nil {
			return nil, fmt.Errorf("list files: %w", err)
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

	infos, ok := result.([]*Info)
	if !ok {
		return nil, fmt.Errorf("cache type mismatch for dir %q", p)
	}
	return infos, nil
}

func (d *Drive) ServeContent(w http.ResponseWriter, r *http.Request, info *Info) error {
	if info.PickCode == "" {
		return errors.New("pick code not found")
	}

	result, err := d.fetchCache(d.cacheKeyDownload(info.PickCode), func() (any, error) {
		var rawURL string
		if err := d.checkRateLimit(r.Context(), func() error {
			dl, err := d.client.Download(info.PickCode)
			if err != nil {
				return fmt.Errorf("download: %w", err)
			}
			rawURL = dl.Url.Url
			return nil
		}); err != nil {
			return nil, err
		}
		return rawURL, nil
	})
	if err != nil {
		return err
	}

	rawURL, ok := result.(string)
	if !ok {
		return fmt.Errorf("cache type mismatch for pick code %q", info.PickCode)
	}
	du, err := url.Parse(rawURL)
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

	r = r.WithContext(context.WithValue(r.Context(), pickCodeCtxKey, info.PickCode))
	r.URL = du
	r.Host = du.Host
	d.reverseProxy.ServeHTTP(w, r)

	return nil
}
