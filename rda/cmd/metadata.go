// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// metadataCmd represents the metadata command
var metadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "return the metadata for the provided graph and node",
	Long:  `rda metadata <graph-id> <node-id>`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		graphID, nodeID := args[0], args[1]

		config, err := newConfig()
		if err != nil {
			return err
		}

		md := Metadata(graphID, nodeID, config)

		if err := json.NewEncoder(os.Stdout).Encode(md); err != nil {
			log.Fatalf("failed streaming response, err: %+v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(metadataCmd)
}
