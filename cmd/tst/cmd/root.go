package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/carzil/tst/internal"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Short: "Backup utility under the hood using rclone.",
	Run:   func(cmd *cobra.Command, args []string) {},
}

func requireCollection(f func(c *internal.Collection, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		c, err := internal.CollectionFromCurrentDir()
		if err != nil {
			log.Fatalf("cannot find current collection: %s", err)
		}
		if err := f(c, cmd, args); err != nil {
			log.Fatalf("error: %s", err)
		}
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
