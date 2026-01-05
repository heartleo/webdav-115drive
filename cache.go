package main

import (
	"context"
	"errors"
	"time"

	"github.com/patrickmn/go-cache"
)

func (d *Drive) cacheKeyDir(path string) string {
	return "dir:" + path
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

func (d *Drive) fetchCache(ctx context.Context, key string, fn func() (any, error)) (any, error) {
	if cached, ok := d.cache.Get(key); ok {
		return cached, nil
	}

	var result any

	err := d.checkRateLimit(ctx, func() error {
		var e error
		result, e = fn()
		return e
	})
	if err != nil {
		return nil, err
	}

	d.cache.Set(key, result, cache.DefaultExpiration)

	return result, nil
}
