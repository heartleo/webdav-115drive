package drive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/patrickmn/go-cache"
)

func TestServeContentRedirectUsesRequestUserAgentWithoutCaching(t *testing.T) {
	const (
		clientUA = "WebDAV-Test/1.0"
		rawURL   = "https://cdn.example.test/video.mkv?token=secret"
	)

	calls := 0
	d := &Drive{
		downloadStrategy: downloadStrategyRedirect,
		downloadURL: func(pickCode, userAgent string) (string, error) {
			calls++
			if pickCode != "pick-code" {
				t.Fatalf("pickCode = %q, want pick-code", pickCode)
			}
			if userAgent != clientUA {
				t.Fatalf("userAgent = %q, want %q", userAgent, clientUA)
			}
			return rawURL, nil
		},
	}
	info := &Info{Path: "/video.mkv", Name: "video.mkv", PickCode: "pick-code"}

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		req := httptest.NewRequest(method, "http://webdav.example.test/dav/video.mkv", nil)
		req.Header.Set("User-Agent", clientUA)
		resp := httptest.NewRecorder()

		if err := d.ServeContent(resp, req, info); err != nil {
			t.Fatalf("ServeContent(%s) error = %v", method, err)
		}
		if got := resp.Code; got != http.StatusFound {
			t.Errorf("ServeContent(%s) status = %d, want %d", method, got, http.StatusFound)
		}
		if got := resp.Header().Get("Location"); got != rawURL {
			t.Errorf("ServeContent(%s) Location = %q, want %q", method, got, rawURL)
		}
	}

	if calls != 2 {
		t.Fatalf("download URL calls = %d, want 2", calls)
	}
}

func TestGetDownloadURLCachesProxyURL(t *testing.T) {
	calls := 0
	d := &Drive{
		cache: cache.New(time.Minute, 2*time.Minute),
		downloadURL: func(_ string, userAgent string) (string, error) {
			calls++
			if userAgent != driver.UA115Browser {
				t.Fatalf("userAgent = %q, want %q", userAgent, driver.UA115Browser)
			}
			return "https://cdn.example.test/file", nil
		},
	}

	for range 2 {
		got, err := d.getCachedDownloadURL(context.Background(), "pick-code", driver.UA115Browser)
		if err != nil {
			t.Fatalf("getDownloadURL() error = %v", err)
		}
		if got != "https://cdn.example.test/file" {
			t.Fatalf("getDownloadURL() = %q", got)
		}
	}

	if calls != 1 {
		t.Fatalf("download URL calls = %d, want 1", calls)
	}
}

func TestOptionsValidateRejectsUnknownDownloadStrategy(t *testing.T) {
	opts := Options{
		UID:              "uid",
		CID:              "cid",
		SEID:             "seid",
		KID:              "kid",
		Rate:             1,
		CacheExpire:      1,
		DownloadStrategy: "invalid",
	}

	if err := opts.validate(); err == nil {
		t.Fatal("validate() error = nil, want unknown strategy error")
	}
}
