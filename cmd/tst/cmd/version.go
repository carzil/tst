package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/carzil/tst/internal"
	"github.com/rclone/rclone/fs"
	"github.com/spf13/cobra"
)

func version(c *internal.Collection, cmd *cobra.Command, args []string) error {
	ver, chunkSet, err := c.MakeVersion()
	if err != nil {
		return err
	}
	ver.Message = args[0]

	remoteFs, err := c.OpenRemote()
	if err != nil {
		return err
	}

	// Push chunk set.
	for hash, chunk := range chunkSet {
		_, err := remoteFs.NewObject(context.Background(), c.BlobRemoteName(hash))
		if err == nil {
			continue
		}
		if err != fs.ErrorObjectNotFound {
			return fmt.Errorf("cannot upload chunk %s: %s", hash, err)
		}

		lc, err := c.ReadChunk(chunk)
		if err != nil {
			return fmt.Errorf("cannot read chunk: %s", err)
		}

		_, err = remoteFs.Put(context.Background(), bytes.NewReader(lc.Data), lc)
		if err != nil {
			return fmt.Errorf("put chunk failed: %s", err)
		}
	}

	// Update current meta db.
	if err := c.StoreVersion(ver); err != nil {
		return fmt.Errorf("cannot update meta db: %s", err)
	}

	// Update remote meta db.
	if err := c.PushMeta(context.Background(), remoteFs); err != nil {
		return fmt.Errorf("cannot push changes to remote: %s", err)
	}

	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version MESSAGE",
	Short: "Creates a new version of current collection and pushes it.",
	Args:  cobra.MinimumNArgs(1),
	Run:   requireCollection(version),
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
