package cmd

import (
	"log"

	"github.com/carzil/tst/internal"
	"github.com/spf13/cobra"
)

func diff(c *internal.Collection, cmd *cobra.Command, args []string) error {
	vers, err := c.GetAllVersions()
	if err != nil {
		return err
	}
	diffFiles, err := c.Diff(vers[len(vers)-1])
	if err != nil {
		return err
	}
	for _, diff := range diffFiles {
		switch diff.Status {
		case internal.StatusDeleted:
			log.Printf("deleted: %s", diff.ObjectName)
		case internal.StatusModified:
			log.Printf("modified: %s", diff.ObjectName)
		case internal.StatusNew:
			log.Printf("unversioned: %s", diff.ObjectName)
		}
	}
	return nil
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Shows new, deleted and modified files since last version.",
	Run:   requireCollection(diff),
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
