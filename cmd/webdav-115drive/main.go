package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/heartleo/webdav-115drive/internal/config"
	"github.com/heartleo/webdav-115drive/internal/drive"
	"github.com/heartleo/webdav-115drive/internal/handler"
)

var configPath = flag.String("config", "./", "config file path")

func main() {
	flag.Parse()

	conf, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config failed", slog.Any("error", err))
		os.Exit(1)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(conf.Server.LogLevel)); err != nil {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	fs, err := drive.New(drive.Options{
		UID:         conf.Drive.UID,
		CID:         conf.Drive.CID,
		SEID:        conf.Drive.SEID,
		KID:         conf.Drive.KID,
		Rate:        conf.Drive.Rate,
		CacheExpire: conf.Drive.CacheExpire,
	})
	if err != nil {
		slog.Error("create drive failed", slog.Any("error", err))
		os.Exit(1)
	}

	h := &handler.Handler{
		FS:       fs,
		BasePath: strings.TrimRight(conf.Server.Path, "/"),
	}

	mux := http.NewServeMux()
	if h.BasePath == "" {
		mux.Handle("/", h)
	} else {
		mux.Handle(h.BasePath+"/", h)
	}

	var httpHandler http.Handler = handler.LogMiddleware(mux)

	if conf.Server.User != "" && conf.Server.Password != "" {
		httpHandler = handler.BasicAuthMiddleware(httpHandler, conf.Server.User, conf.Server.Password)
	}

	addr := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("webdav serve",
		slog.String("path", h.BasePath),
		slog.String("addr", addr),
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("webdav serve failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", slog.Any("error", err))
	}
}
