package internal

import (
	"context"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/hash"
)

type LoadedChunk struct {
	Data       []byte
	remoteName string
}

func (lc LoadedChunk) Fs() fs.Info {
	return nil
}

func (lc LoadedChunk) Hash(ctx context.Context, ty hash.Type) (string, error) {
	return "", nil
}

func (lc LoadedChunk) Storable() bool {
	return true
}

func (lc LoadedChunk) String() string {
	return "loaded chunk [remote: " + lc.remoteName + "]"
}

func (lc LoadedChunk) Remote() string {
	return lc.remoteName
}

func (lc LoadedChunk) ModTime(context.Context) time.Time {
	return time.Now()
}

func (lc LoadedChunk) Size() int64 {
	return int64(len(lc.Data))
}
