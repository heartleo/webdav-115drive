package drive

import (
	"context"
	"net/http"
	"time"
)

type FileSystem interface {
	Stat(ctx context.Context, path string) (*Info, error)
	ReadDir(ctx context.Context, path string) ([]*Info, error)
}

// ContentServer is implemented by backends that can proxy file downloads directly,
// avoiding an intermediate buffer. Handler checks for this via interface assertion
// rather than a concrete type dependency.
type ContentServer interface {
	ServeContent(w http.ResponseWriter, r *http.Request, info *Info) error
}

type Info struct {
	Path     string
	Name     string
	IsDir    bool
	Size     int64
	ModTime  time.Time
	ETag     string
	PickCode string
}
