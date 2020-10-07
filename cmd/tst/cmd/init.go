package cmd

import (
	"context"
	"log"

	"github.com/carzil/mipt-testing-2020/internal"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init REMOTE",
	Short: "Initialize an empty collection at current directory.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := internal.CreateCollectionAt(".", args[0]); err != nil {
			log.Fatalf("cannot create collection: %s", err)
		}

		c, err := internal.CollectionFromCurrentDir()
		if err != nil {
			log.Fatalf("cannot open newly created collection: %s", err)
		}

		if err := c.RestoreLastVersion(context.Background()); err != nil {
			log.Fatalf("cannot restore last version: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
