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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/spf13/cobra"
)

// dg1bCmd represents the dg1b command
var dg1bCmd = &cobra.Command{
	Use:   "dg1b",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("dg1b called")
	// },
}

var dg1bMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "metadata describing the 1B image",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dg1b called")
	},
}

var dg1bPartsCmd = &cobra.Command{
	Use:   "parts",
	Short: "returns a description of the image parts that compose the 1B image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, writeConfig, err := newClient(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err := writeConfig(); err != nil {
				log.Printf("on exit, received an error when writing configuration, err: %v", err)
			}
		}()

		parts, err := rda.PartSummary(client, args[0])
		if err != nil {
			return err
		}

		type BandSummary struct {
			NumParts int      `json:"numParts"`
			ImageIDs []string `json:"imageIDs"`
		}
		summary := struct {
			Cavis *BandSummary `json:"cavis,omitempty"`
			Pan   *BandSummary `json:"pan,omitempty"`
			NVIR  *BandSummary `json:"nvir,omitempty"`
			SWIR  *BandSummary `json:"swir,omitempty"`
		}{}
		for _, bandType := range []struct {
			bs    **BandSummary
			parts []rda.ImageMetadata
		}{
			{&summary.Cavis, parts.CavisImages},
			{&summary.Pan, parts.PanImages},
			{&summary.NVIR, parts.VNIRImages},
			{&summary.SWIR, parts.SWIRImages},
		} {
			if len(bandType.parts) == 0 {
				continue
			}
			bs := BandSummary{NumParts: len(bandType.parts)}
			for _, part := range bandType.parts {
				bs.ImageIDs = append(bs.ImageIDs, part.ImageID)
			}
			*bandType.bs = &bs
		}
		return json.NewEncoder(os.Stdout).Encode(&summary)
	},
}

var dg1bRealizeCmd = &cobra.Command{
	Use:   "realize",
	Short: "realize the 1B image",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("realize called")
	},
}

func init() {
	rootCmd.AddCommand(dg1bCmd)
	dg1bCmd.AddCommand(dg1bMetadataCmd)
	dg1bCmd.AddCommand(dg1bPartsCmd)
	dg1bCmd.AddCommand(dg1bRealizeCmd)
}
