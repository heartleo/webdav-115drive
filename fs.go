package main

import (
	"context"
	"io"
	"time"
)

type FS interface {
	Stat(ctx context.Context, p string) (*Info, error)
	ReadDir(ctx context.Context, p string) ([]*Info, error)
	Open(ctx context.Context, p string) (io.ReadSeeker, *Info, error)
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
