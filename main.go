package main

import (
	"errors"
	"flag"
	"fmt"
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
	configPath = flag.String("config", "./", "config file path")
)

func main() {
	flag.Parse()

	conf, err := loadConfig(*configPath)
	if err != nil {
		slog.Error("load config failed", slog.Any("error", err))
		os.Exit(1)
	}

	fs, err := newDrive(&conf.Drive)
	if err != nil {
		slog.Error("create drive failed", slog.Any("error", err))
		os.Exit(1)
	}

	h := &Handler{
		FS:       fs,
		BasePath: strings.TrimRight(conf.Server.Path, "/"),
	}

	mux := http.NewServeMux()
	if h.BasePath == "/" {
		mux.Handle("/", h)
	} else {
		mux.Handle(h.BasePath+"/", h)
	}

	handler := logMiddleware(mux)

	if conf.Server.User != "" && conf.Server.Pwd != "" {
		handler = basicAuthMiddleware(handler, conf.Server.User, conf.Server.Pwd)
	}

	addr := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("webdav serve", slog.String("path", h.BasePath), slog.String("addr", addr))

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("webdav serve failed", slog.Any("error", err))
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

func basicAuthMiddleware(next http.Handler, user, pwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pwd {
			w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
