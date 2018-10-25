// Copyright © 2018 DigitalGlobe
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
	Hidden: true,
	Use:    "metadata <rda-graph-id> <rda-node-id>",
	Short:  "return the metadata for the provided RDA graph and node",
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

		md, err := rda.GraphMetadata(graphID, nodeID, client)
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(md)
	},
}

func init() {
	rootCmd.AddCommand(metadataCmd)
}
