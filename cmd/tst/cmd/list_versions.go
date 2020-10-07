package cmd

import (
	"log"

	"github.com/carzil/tst/internal"
	"github.com/spf13/cobra"
)

func listVersions(c *internal.Collection, cmd *cobra.Command, args []string) error {
	vers, err := c.GetAllVersions()
	if err != nil {
		return err
	}
	for v, ver := range vers {
		log.Printf("v%d | %s", v, ver.Message)
	}
	return nil
}

var listVersionsCmd = &cobra.Command{
	Use:   "list-versions",
	Short: "List all versions for current collection.",
	Run:   requireCollection(listVersions),
}

func init() {
	rootCmd.AddCommand(listVersionsCmd)
}
