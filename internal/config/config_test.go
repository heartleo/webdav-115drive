package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDownloadStrategy(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("DRIVE_DOWNLOAD_STRATEGY", "")

		conf, err := Load(t.TempDir())
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got := conf.Drive.DownloadStrategy; got != "redirect" {
			t.Fatalf("DownloadStrategy = %q, want redirect", got)
		}
	})

	t.Run("configured", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("drive:\n  download_strategy: proxy\n"), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		conf, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got := conf.Drive.DownloadStrategy; got != "proxy" {
			t.Fatalf("DownloadStrategy = %q, want proxy", got)
		}
	})
}
