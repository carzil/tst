package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/carzil/tst/internal"
	"github.com/spf13/cobra"
)

func restore(c *internal.Collection, cmd *cobra.Command, args []string) error {
	toVer, err := strconv.Atoi(args[0][1:])
	if err != nil {
		return fmt.Errorf("invalid version reference '%s' (must be v0, v1, etc)", args[0])
	}
	if !confirm("Restore will replace files in current collection and thus may wipe unversioned data. Continue?") {
		return nil
	}
	return c.RestoreVersion(context.Background(), toVer, c.Root)
}

var restoreCmd = &cobra.Command{
	Use:   "restore VERSION",
	Short: "Restores specified version.",
	Args:  cobra.ExactArgs(1),
	Run:   requireCollection(restore),
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
