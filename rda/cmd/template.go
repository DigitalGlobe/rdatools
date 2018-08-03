package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Hidden: true,
	Use:    "template <template-id>",
	Short:  "Returns metadata describing the template's graph.",
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := newConfig()
		if err != nil {
			return err
		}

		client, ts, err := newClient(context.TODO(), &config)
		if err != nil {
			return err
		}
		defer writeConfig(&config, ts)

		md, err := rda.TemplateMetadata("DigitalGlobeStrip", client, nil)

		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(md)
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// templateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// templateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
