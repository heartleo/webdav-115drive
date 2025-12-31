package main

import (
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func init() {
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	}))
	slog.SetDefault(l)
}

var (
	listen     = flag.String("listen", ":8090", "listen address")
	basePath   = flag.String("tripper", "/dav", "url tripper path (e.g. /dav)")
	configPath = flag.String("config", "./", "config file path")
)

func main() {
	flag.Parse()

	conf, err := LoadConfig(*configPath)
	if err != nil {
		slog.Error("load config failed", slog.Any("error", err))
		os.Exit(1)
	}

	fs, err := NewDrive(&conf.Drive)
	if err != nil {
		slog.Error("create drive failed", slog.Any("error", err))
		os.Exit(1)
	}

	h := &Handler{
		FS:       fs,
		BasePath: strings.TrimRight(*basePath, "/"),
	}

	mux := http.NewServeMux()
	mux.Handle(h.BasePath+"/", h)
	mux.Handle(h.BasePath, http.RedirectHandler(h.BasePath+"/", http.StatusMovedPermanently))

	srv := &http.Server{
		Addr:              *listen,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("WebDAV server starting",
		slog.String("listen", *listen),
		slog.String("tripper", h.BasePath),
	)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("serve failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
