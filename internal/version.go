package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

type File struct {
	Name   string   `json:"-"`
	Chunks []string `json:"chunks"`
}

type Version struct {
	Message string
	Files   map[string]File
}

func NewVersionFromBucket(b *bbolt.Bucket) (Version, error) {
	ver := Version{
		Files: make(map[string]File),
	}

	cursor := b.Cursor()
	for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
		if k[0] == 0 {
			// Control key.
			switch string(k[1:]) {
			case "message":
				ver.Message = string(v)
			default:
			}
		} else {
			// File state.
			f := File{Name: string(k)}
			if err := json.Unmarshal(v, &f); err != nil {
				return Version{}, fmt.Errorf("invalid state for file %s: %s", string(k), err)
			}
			ver.Files[f.Name] = f
		}
	}

	return ver, nil
}

func (ver Version) Save(b *bbolt.Bucket) error {
	if err := b.Put([]byte("\x00message"), []byte(ver.Message)); err != nil {
		return err
	}

	for name, object := range ver.Files {
		data, err := json.Marshal(&object)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(name), data); err != nil {
			return err
		}
	}

	return nil
}

type Chunk struct {
	ParentObject   string
	ParentPosition int64
	Hash           string
}

type ChunkSet map[string]Chunk

func (c Collection) MakeVersion() (Version, ChunkSet, error) {
	ver := Version{
		Files: make(map[string]File),
	}
	chunkSet := make(ChunkSet)

	err := filepath.Walk(c.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		objectName, err := filepath.Rel(c.Root, path)
		if err != nil {
			return err
		}

		if objectName == ".collection" {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		chunks, err := FileChunkedChecksums(f)
		if err != nil {
			return err
		}

		for idx, chunk := range chunks {
			if _, ok := chunkSet[chunk]; !ok {
				chunkSet[chunk] = Chunk{
					ParentObject:   objectName,
					ParentPosition: chunkSize * int64(idx),
					Hash:           chunk,
				}
			}
		}

		ver.Files[objectName] = File{
			Name:   objectName,
			Chunks: chunks,
		}

		return nil
	})

	if err != nil {
		return Version{}, ChunkSet{}, err
	}

	return ver, chunkSet, nil
}

func (c Collection) ReadChunk(ch Chunk) (LoadedChunk, error) {
	f, err := os.Open(filepath.Join(c.Root, ch.ParentObject))
	if err != nil {
		return LoadedChunk{}, err
	}
	defer f.Close()

	lc := LoadedChunk{}
	lc.Data = make([]byte, chunkSize)
	lc.remoteName = c.BlobRemoteName(ch.Hash)
	n, err := f.ReadAt(lc.Data, ch.ParentPosition)
	if err != nil && err != io.EOF {
		return LoadedChunk{}, err
	}
	lc.Data = lc.Data[:n]
	return lc, nil
}
