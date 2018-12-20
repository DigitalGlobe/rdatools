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
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// dg1bTemplateID rda template created by uploading the idaho read operator to RDA
const dg1bTemplateID = "848c481257a100ae373523df9f23c0176484b6f63757e9e58d2fa9c2d2af12d9"

// dg1bCmd represents the dg1b command
var dg1bCmd = &cobra.Command{
	Use:   "dg1b",
	Short: "commands to access DigitalGlobe 1Bs from RDA",
	//Hidden: true,
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("dg1b called")
	// },
}

var dg1bMetadataCmd = &cobra.Command{
	Use:   "metadata <catalog id> <band> <part number>",
	Short: "metadata describing the 1B image part",
	Long: `metadata describing the 1B image part

You must provide the catalog id, band (e.g. pan, vnir, swir, or
cavis), and part number to get (starting at 1), in that order. Use the
"dg1b parts" command to figure out valid bands and part numbers.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		// The http client.
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

		// Parse the args.
		catID, bandName := args[0], args[1]
		partNum, err := strconv.Atoi(args[2])
		if err != nil {
			return errors.Errorf("part number %q cannot be converted to an integer", args[2])
		}
		if partNum < 1 {
			return errors.New("part numbers start at 1")
		}
		partNum--
		bandName = strings.ToLower(bandName)

		// Go find the rda image id associated with this part.
		parts, err := rda.PartSummary(client, catID)
		if err != nil {
			return err
		}

		var images []rda.ImageMetadata
		switch bandName {
		case "pan":
			images = parts.PanImages
		case "vnir":
			images = parts.VNIRImages
		case "swir":
			images = parts.SWIRImages
		case "cavis":
			images = parts.CavisImages
		default:
			return errors.Errorf("band argument %q is not of type pan, vnir, swir, or cavis", bandName)
		}
		if partNum >= len(images) {
			return errors.Errorf("band %q has %d parts", bandName, len(images))
		}
		imageMD := images[partNum]

		// Get the metadata.
		template := rda.NewTemplate(dg1bTemplateID, client,
			rda.AddParameter("imageId", imageMD.ImageID),
			rda.AddParameter("bucketName", imageMD.TileBucketName))
		md, err := template.Metadata()
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(md)

	},
}

var dg1bPartsCmd = &cobra.Command{
	Use:   "parts <catalog id>",
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
			VNIR  *BandSummary `json:"vnir,omitempty"`
			SWIR  *BandSummary `json:"swir,omitempty"`
		}{}
		for _, bandType := range []struct {
			bs    **BandSummary
			parts []rda.ImageMetadata
		}{
			{&summary.Cavis, parts.CavisImages},
			{&summary.Pan, parts.PanImages},
			{&summary.VNIR, parts.VNIRImages},
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
	Use:   "realize <catalog id> <band> <part number> <outdir>",
	Short: "realize a 1B image part from RDA",
	Long: `realize a 1B image part from RDA

You must provide the catalog id, band (e.g. pan, vnir, swir, or
cavis), part number to get (starting at 1), and output directory. Use the
"dg1b parts" command to figure out valid bands and part numbers.`,

	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		// The http client.
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

		// Parse the args.
		catID, bandName := args[0], args[1]
		partNum, err := strconv.Atoi(args[2])
		if err != nil {
			return errors.Errorf("part number %q cannot be converted to an integer", args[2])
		}
		if partNum < 1 {
			return errors.New("part numbers start at 1")
		}
		partNum--
		bandName = strings.ToLower(bandName)
		outDir := args[3]

		// Go find the rda image id associated with this part, building the metadata prefix while we're at it.
		parts, err := rda.PartSummary(client, catID)
		if err != nil {
			return err
		}

		var images []rda.ImageMetadata
		var partPrefix string
		switch bandName {
		case "pan":
			images = parts.PanImages
			partPrefix = fmt.Sprintf("PAN_P%03d", partNum+1)
		case "vnir":
			images = parts.VNIRImages
			partPrefix = fmt.Sprintf("MUL_P%03d", partNum+1)
		case "swir":
			images = parts.SWIRImages
			partPrefix = fmt.Sprintf("SWIR_P%03d", partNum+1)
		case "cavis":
			images = parts.CavisImages
			partPrefix = fmt.Sprintf("CAVIS_P%03d", partNum+1)
		default:
			return errors.Errorf("band argument %q is not of type pan, vnir, swir, or cavis", bandName)
		}
		if partNum >= len(images) {
			return errors.Errorf("band %q has %d parts", bandName, len(images))
		}

		// Download the metadata and extract the relevent files to outDir.
		rpcs, err := rda.PartMetadata(client, catID, partPrefix, outDir)
		if err != nil {
			return err
		}

		// Get the RDA metadata.
		imageMD := images[partNum]
		template := rda.NewTemplate(dg1bTemplateID, client,
			rda.AddParameter("imageId", imageMD.ImageID),
			rda.AddParameter("bucketName", imageMD.TileBucketName))
		md, err := template.Metadata()
		if err != nil {
			return err
		}
		rda.WithWindow(md.ImageMetadata.TileWindow)(template)

		// Download the tiles.
		bar := pb.StartNew(md.ImageMetadata.NumXTiles * md.ImageMetadata.NumYTiles)
		rda.WithProgressFunc(bar.Increment)(template)

		tileDir := filepath.Join(outDir, "tiles")
		tStart := time.Now()
		tiles, err := template.Realize(ctx, tileDir)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			bar.FinishPrint(fmt.Sprintf("Completed %d of %d 1B tiles before cancellation; rerun the command to pick up where you left off.", len(tiles), md.ImageMetadata.NumXTiles*md.ImageMetadata.NumYTiles))
		default:
			bar.FinishPrint(fmt.Sprintf("Tile retrieval took %s", time.Since(tStart)))
		}
		if len(tiles) < 1 {
			return err
		}

		// Build VRT struct and write it to disk.
		vrt, err := rda.NewVRT(md, tiles, rpcs)
		if err != nil {
			return err
		}

		vrtPath := filepath.Join(outDir, partPrefix+".vrt")
		f, err := os.Create(vrtPath)
		if err != nil {
			return errors.Wrap(err, "failed creating VRT for downloaded tiles")
		}
		defer f.Close()

		if err := vrt.MakeRelative(filepath.Dir(vrtPath)); err != nil {
			return err
		}

		enc := xml.NewEncoder(f)
		enc.Indent("  ", "    ")
		if err := enc.Encode(vrt); err != nil {
			return errors.Wrap(err, "couldn't write our VRT to disk")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dg1bCmd)
	dg1bCmd.AddCommand(dg1bMetadataCmd)
	dg1bCmd.AddCommand(dg1bPartsCmd)
	dg1bCmd.AddCommand(dg1bRealizeCmd)
}
