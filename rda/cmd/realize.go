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
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/cheggaaa/pb"
	"github.com/spf13/cobra"
)

// realizeCmd represents the realize command
var realizeCmd = &cobra.Command{
	Use:   "realize",
	Short: "Materialize the tiles that compose a graph and wrap it in a VRT",
	//Long: ``,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		log.SetFlags(log.Lshortfile)

		graphID, nodeID := args[0], args[1]
		shortGraphID := graphID
		if len(shortGraphID) > 10 {
			shortGraphID = shortGraphID[0:10]
		}

		config := NewConfig()

		md := Metadata(graphID, nodeID, config)

		// md.ImageMetadata.NumXTiles = 3
		// md.ImageMetadata.NumYTiles = 3

		// Download tiles.
		r, err := rda.NewRetriever(graphID, nodeID, *md, fmt.Sprintf("%s-%s", shortGraphID, nodeID), config.Token)
		if err != nil {
			log.Fatal(err)
		}

		bar := pb.StartNew(md.ImageMetadata.NumXTiles * md.ImageMetadata.NumYTiles)

		ts := time.Now()
		tileMap := r.Retrieve(bar.Increment)
		elapsed := time.Since(ts)
		bar.FinishPrint("The End!")
		log.Printf("Tile retrieval took %s", elapsed)

		// Build VRT.
		vrt, err := rda.NewVRT(md, tileMap)
		if err != nil {
			log.Fatal("failed building vrt")
		}

		vrtFile := fmt.Sprintf("%s-%s.vrt", shortGraphID, nodeID)
		f, err := os.Create(vrtFile)
		if err != nil {
			log.Fatal("failure creating VRT file")
		}
		defer f.Close()

		enc := xml.NewEncoder(f)
		enc.Indent("  ", "    ")
		if err := enc.Encode(vrt); err != nil {
			log.Fatalf("error: %v\n", err)
		}

		fmt.Printf("VRT available at %s\n", vrtFile)
	},
}

func init() {
	rootCmd.AddCommand(realizeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// realizeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// realizeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
