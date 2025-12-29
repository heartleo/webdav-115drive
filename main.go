package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		listen     = flag.String("listen", ":8090", "listen address")
		basePath   = flag.String("base", "/dav", "url base path (e.g. /dav)")
		configPath = flag.String("config", "./", "config file path (optional)")
	)
	flag.Parse()

	// Load config
	conf, err := LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	// Create 115 drive file system
	fs, err := NewDrive115FS(conf.Drive115)
	if err != nil {
		slog.Error("failed to create drive115 fs", slog.Any("error", err))
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

	slog.Info("115 WebDAV server starting",
		slog.String("listen", *listen),
		slog.String("base", h.BasePath),
	)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("error", err))
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
