package cmd

import (
	"encoding/json"
	"os"

	"context"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/spf13/cobra"
)

// metadataCmd represents the metadata command
var metadataCmd = &cobra.Command{
	Use:   "metadata <rda-graph-id> <rda-node-id>",
	Short: "return the metadata for the provided RDA graph and node",
	//Long:  `return the metadata for the provided RDA graph and node`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		graphID, nodeID := args[0], args[1]
		config, err := newConfig()
		if err != nil {
			return err
		}

		client, ts, err := newClient(context.TODO(), &config)
		if err != nil {
			return err
		}
		defer writeConfig(&config, ts)

		md, err := rda.FetchMetadata(graphID, nodeID, client)
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(md)
	},
}

func init() {
	rootCmd.AddCommand(metadataCmd)
}
