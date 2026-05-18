package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	webdav115drive "github.com/heartleo/webdav-115drive"
	"github.com/heartleo/webdav-115drive/internal/config"
	"github.com/heartleo/webdav-115drive/internal/drive"
	"github.com/heartleo/webdav-115drive/internal/handler"
	"github.com/heartleo/webdav-115drive/internal/tui"
	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:           "webdav-115drive",
	Short:         "A read-only WebDAV proxy for 115 Drive",
	Version:       webdav115drive.Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s version %s (commit %s, built %s)\n",
			rootCmd.Name(), webdav115drive.Version, webdav115drive.Commit, webdav115drive.Date)
		return err
	},
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Browse the configured 115 Drive in an interactive terminal UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "./", "config file path")
	rootCmd.SetVersionTemplate("{{printf \"%s version %s\\n\" .Name .Version}}")
	if f := rootCmd.Flags().Lookup("version"); f != nil {
		f.Shorthand = "v"
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(tuiCmd)
}

func runTUI() error {
	conf, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Silence slog: TUI takes over the alt-screen and stdout writes would
	// corrupt the rendering. Errors are surfaced via the model + return value.
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	fs, err := drive.New(drive.Options{
		UID:         conf.Drive.UID,
		CID:         conf.Drive.CID,
		SEID:        conf.Drive.SEID,
		KID:         conf.Drive.KID,
		Rate:        conf.Drive.Rate,
		CacheExpire: conf.Drive.CacheExpire,
	})
	if err != nil {
		return fmt.Errorf("create drive: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return tui.Run(ctx, fs)
}

func runServe() error {
	conf, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
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
		return fmt.Errorf("create drive: %w", err)
	}

	h, err := handler.New(fs, conf.Server.Path)
	if err != nil {
		return fmt.Errorf("create handler: %w", err)
	}

	mux := http.NewServeMux()
	if h.BasePath == "" {
		mux.Handle("/", h)
	} else {
		mux.Handle(h.BasePath+"/", h)
	}

	httpHandler := handler.LogMiddleware(mux)
	if conf.Server.User != "" && conf.Server.Password != "" {
		httpHandler = handler.BasicAuthMiddleware(httpHandler, conf.Server.User, conf.Server.Password)
	}

	addr := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	slog.Info("webdav serve",
		slog.String("path", h.BasePath),
		slog.String("addr", addr),
	)

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("webdav serve: %w", err)
	case <-quit:
	}

	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", slog.Any("error", err))
	}
	return nil
}
