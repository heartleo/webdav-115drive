package drive

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/patrickmn/go-cache"
)

func (d *Drive) cacheKeyDir(p string) string {
	return "dir:" + p
}

func (d *Drive) cacheKeyDownload(pickCode string) string {
	return "download:" + pickCode
}

func (d *Drive) checkRateLimit(ctx context.Context, fn func() error) error {
	if d.limiter != nil {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := d.limiter.Wait(ctx); err != nil {
			return errors.New("rate limit exceeded")
		}
	}
	return fn()
}

// fetchCache does not apply rate limiting; callers are responsible for
// rate-limiting fn to avoid double-counting tokens.
func (d *Drive) fetchCache(key string, fn func() (any, error)) (any, error) {
	if cached, ok := d.cache.Get(key); ok {
		slog.Debug("cache hit", slog.String("key", key))
		return cached, nil
	}

	slog.Debug("cache miss", slog.String("key", key))

	result, err, _ := d.group.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		return nil, err
	}

	d.cache.Set(key, result, cache.DefaultExpiration)
	return result, nil
}
