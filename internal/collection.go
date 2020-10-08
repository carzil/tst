package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/cache"
	"go.etcd.io/bbolt"
)

var metaBucket = []byte("meta")
var checksumsBucket = []byte("checksums")

type Collection struct {
	db         *bbolt.DB
	Root       string
	remoteName string
	RemoteRoot string
}

func (c Collection) OpenRemote() (fs.Fs, error) {
	fs, _, err := openRemote(c.remoteName)
	return fs, err
}

func (c Collection) StoreVersion(ver Version) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		meta := tx.Bucket(metaBucket)
		currVerBs := meta.Get([]byte("version"))
		newVer := 0
		if currVerBs != nil {
			var err error
			newVer, err = strconv.Atoi(string(currVerBs))
			if err != nil {
				return err
			}
			newVer++
		}
		b, err := tx.CreateBucket([]byte(fmt.Sprintf("ver%d", newVer)))
		if err != nil {
			return err
		}
		if err := ver.Save(b); err != nil {
			return err
		}
		if err := meta.Put([]byte("version"), []byte(strconv.Itoa(newVer))); err != nil {
			return err
		}
		return nil
	})
}

func (c Collection) PushMeta(ctx context.Context, remoteFs fs.Fs) error {
	return c.db.View(func(tx *bbolt.Tx) error {
		b := &bytes.Buffer{}
		if _, err := tx.WriteTo(b); err != nil {
			return err
		}

		lc := LoadedChunk{
			remoteName: c.RemoteRoot + "/db",
			Data:       b.Bytes(),
		}

		if _, err := remoteFs.Put(ctx, bytes.NewReader(lc.Data), lc); err != nil {
			return err
		}

		return nil
	})
}

func (c Collection) BlobRemoteName(hash string) string {
	return c.RemoteRoot + "/blobs/" + hash
}

func (c Collection) restoreObject(ctx context.Context, remoteFs fs.Fs, objectName string, object File) error {
	f, err := os.OpenFile(filepath.Join(c.Root, objectName), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, hash := range object.Chunks {
		remoteBlob, err := remoteFs.NewObject(ctx, c.BlobRemoteName(hash))
		if err != nil {
			return err
		}
		r, err := remoteBlob.Open(ctx)
		if err != nil {
			return err
		}
		if _, err = io.Copy(f, r); err != nil {
			return err
		}
	}

	log.Printf("restored object %s [%d chunks]", objectName, len(object.Chunks))

	return nil

}

func (c Collection) RestoreLastVersion(ctx context.Context) error {
	vers, err := c.GetAllVersions()
	if err != nil {
		return err
	}
	if len(vers) > 0 {
		return c.RestoreVersion(ctx, len(vers)-1, c.Root)
	}
	return nil
}

func (c Collection) RestoreVersion(ctx context.Context, targetVer int, dst string) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(fmt.Sprintf("ver%d", targetVer)))
		if b == nil {
			return fmt.Errorf("no version %d", targetVer)
		}
		ver, err := NewVersionFromBucket(b)
		if err != nil {
			return fmt.Errorf("corrupted meta db: %s", err)
		}

		// Remove unused files.
		err = filepath.Walk(c.Root, func(path string, info os.FileInfo, err error) error {
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

			if _, ok := ver.Files[objectName]; !ok {
				if err := os.Remove(objectName); err != nil {
					return fmt.Errorf("cannot remove %s: %s", objectName, err)
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		remoteFs, err := c.OpenRemote()
		if err != nil {
			return fmt.Errorf("cannot open remote: %s", err)
		}

		// Restore new versions.
		for objectName, object := range ver.Files {
			if err := c.restoreObject(ctx, remoteFs, objectName, object); err != nil {
				return fmt.Errorf("cannot restore %s: %s", objectName, err)
			}
		}

		log.Printf("successfully restored v%d [%s]", targetVer, ver.Message)

		return nil
	})
}

const (
	StatusDeleted  = 1
	StatusNew      = 2
	StatusModified = 3
)

type ObjectDiff struct {
	ObjectName string
	Status     int
}

func (c Collection) checkObjectModified(object File) (bool, error) {
	f, err := os.Open(filepath.Join(c.Root, object.Name))
	if err != nil {
		return false, err
	}
	defer f.Close()
	newChunks, err := FileChunkedChecksums(f)
	if err != nil {
		return false, err
	}
	if len(newChunks) != len(object.Chunks) {
		return true, nil
	}
	for idx, newChunk := range newChunks {
		if newChunk != object.Chunks[idx] {
			return true, nil
		}
	}
	return false, nil
}

func (c Collection) Diff(ver Version) ([]ObjectDiff, error) {
	var diff []ObjectDiff
	present := make(map[string]struct{})
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

		object, ok := ver.Files[objectName]
		if !ok {
			diff = append(diff, ObjectDiff{
				Status:     StatusNew,
				ObjectName: objectName,
			})
		} else {
			present[objectName] = struct{}{}
			modified, err := c.checkObjectModified(object)
			if err != nil {
				return err
			}
			if modified {
				diff = append(diff, ObjectDiff{
					Status:     StatusModified,
					ObjectName: objectName,
				})
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(present) != len(ver.Files) {
		for objectName := range ver.Files {
			if _, ok := present[objectName]; !ok {
				diff = append(diff, ObjectDiff{
					Status:     StatusDeleted,
					ObjectName: objectName,
				})
			}
		}
	}
	return diff, nil
}

func openRemote(remoteName string) (fs.Fs, string, error) {
	_, _, remoteRoot, err := fs.ParseRemote(remoteName)
	if err != nil {
		return nil, "", fmt.Errorf("invalid destination '%s': %s", remoteName, err)
	}

	remoteFs, err := cache.Get(remoteName)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open '%s': %s", remoteName, err)
	}

	return remoteFs, remoteRoot, nil
}

func CreateCollectionAt(path string, remoteName string) error {
	remoteFs, remoteRoot, err := openRemote(remoteName)
	if err != nil {
		return err
	}

	var db *bbolt.DB

	dbObj, err := remoteFs.NewObject(context.TODO(), remoteRoot+"/db")
	if err != nil {
		if err != fs.ErrorObjectNotFound {
			return fmt.Errorf("cannot access remote meta db: %s", err)
		}
	} else {
		r, err := dbObj.Open(context.TODO())
		if err != nil {
			return fmt.Errorf("cannot open remote meta db: %s", err)
		}

		f, err := os.Create(filepath.Join(path, ".collection"))
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(f, r); err != nil {
			return fmt.Errorf("cannot copy remote meta db: %s", err)
		}
	}

	db, err = bbolt.Open(filepath.Join(path, ".collection"), 0700, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(checksumsBucket); err != nil {
			return err
		}

		b, err := tx.CreateBucketIfNotExists(metaBucket)
		if err != nil {
			return err
		}

		return b.Put([]byte("remote"), []byte(remoteName))
	})
}

func CollectionFromCurrentDir() (*Collection, error) {
	currPath, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}

	for {
		if _, err := os.Stat(filepath.Join(currPath, ".collection")); err == nil {
			break
		}

		nextPath := filepath.Clean(filepath.Join(currPath, ".."))
		if nextPath == currPath {
			return nil, fmt.Errorf("current directory or any parent doesn't contain a valid collection")
		}
		currPath = nextPath
	}

	db, err := bbolt.Open(filepath.Join(currPath, ".collection"), 0700, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open internal file: %s", err)
	}

	var remoteName string

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(metaBucket)
		if b == nil {
			return fmt.Errorf("invalid db: no meta bucket")
		}
		remoteName = string(b.Get([]byte("remote")))

		return err
	})

	if err != nil {
		return nil, fmt.Errorf("invalid database: %s", err)
	}

	_, _, remoteRoot, err := fs.ParseRemote(remoteName)
	if err != nil {
		return nil, fmt.Errorf("fatal error: invalid destination '%s' from meta bucket: %s", remoteName, err)
	}

	return &Collection{
		db:         db,
		Root:       currPath,
		RemoteRoot: remoteRoot,
		remoteName: remoteName,
	}, nil
}

func (c Collection) GetAllVersions() ([]Version, error) {
	var versions []Version
	err := c.db.View(func(tx *bbolt.Tx) error {
		v := 0
		for {
			b := tx.Bucket([]byte(fmt.Sprintf("ver%d", v)))
			if b == nil {
				return nil
			}
			ver, err := NewVersionFromBucket(b)
			if err != nil {
				return fmt.Errorf("corrupted meta db: invalid version %d: %s", v, err)
			}
			versions = append(versions, ver)
			v++
		}
	})

	if err != nil {
		return nil, err
	}

	return versions, nil
}
