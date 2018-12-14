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
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const dgstripTemplateID = "DigitalGlobeStrip"

// dgstripRealizeCmd represents the dgstrip command
var dgstripCmd = &cobra.Command{
	Use:   "dgstrip",
	Short: "Subcommand for accessing DigitalGlobe image strips from RDA",
}

// dgstripRealizeCmd represents the dgstrip command
var dgstripRealizeCmd = &cobra.Command{
	Use:   "realize <catalog-id> <output-vrt>",
	Short: "Realize tiles of a DigitalGlobe strip from RDA",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Setup our context to handle cancellation and listen for signals.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			select {
			case s := <-sigs:
				log.Printf("received a shutdown signal %s, winding down", s)
				cancel()
			case <-ctx.Done():
			}
		}()

		// The http client.
		client, writeConfig, err := newClient(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err := writeConfig(); err != nil {
				log.Printf("on exit, received an error when writing configuration, err: %v", err)
			}
		}()

		// Get the metadata and figure out what RDA tiles need to be downloaded.
		catID, vrtPath := args[0], args[1]
		template := rda.NewTemplate(dgstripTemplateID, client, dgstripTemplateOptions(catID)...)
		md, err := template.Metadata()
		if err != nil {
			return err
		}
		tileWindow, err := processSubWindows(&dgstripFlags.srcWin, &dgstripFlags.projWin, md)
		if err != nil {
			return err
		}
		rda.WithWindow(*tileWindow)(template)

		// Get the tiles.
		bar := pb.StartNew(tileWindow.NumXTiles * tileWindow.NumYTiles)
		rda.WithProgressFunc(bar.Increment)(template)

		tileDir := vrtPath[:len(vrtPath)-len(path.Ext(vrtPath))]
		tStart := time.Now()
		tiles, err := template.Realize(ctx, tileDir)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			bar.FinishPrint(fmt.Sprintf("Completed %d of %d tiles before cancellation; rerun the command to pick up where you left off.", len(tiles), tileWindow.NumXTiles*tileWindow.NumYTiles))
		default:
			bar.FinishPrint(fmt.Sprintf("Tile retrieval took %s", time.Since(tStart)))
		}
		if len(tiles) < 1 {
			return err
		}

		// Build VRT struct and write it to disk.
		vrt, err := rda.NewVRT(md, tiles, nil)
		if err != nil {
			return err
		}

		f, err := os.Create(vrtPath)
		if err != nil {
			return errors.Wrap(err, "failed creating VRT for downloaded tiles")
		}
		defer f.Close()

		enc := xml.NewEncoder(f)
		enc.Indent("  ", "    ")
		if err := enc.Encode(vrt); err != nil {
			return errors.Wrap(err, "couldn't write our VRT to disk")
		}
		return nil
	},
}

var dgstripBatchCmd = &cobra.Command{
	Use:   "batch <catalog-id>",
	Short: "Realize images of a DigitalGlobe strip from RDA via RDA batch materialization",
	Args:  cobra.ExactArgs(1),
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

		// Get the metadata and figure out what RDA tiles need to be downloaded.
		catID := args[0]
		template := rda.NewTemplate(dgstripTemplateID, client, dgstripTemplateOptions(catID)...)

		// If we were given a subwindow, figure out its
		// mapping to RDA tiles.
		if (dgstripFlags.projWin != projectionWindow{} || dgstripFlags.srcWin != sourceWindow{}) {
			md, err := template.Metadata()
			if err != nil {
				return err
			}
			tileWindow, err := processSubWindows(&dgstripFlags.srcWin, &dgstripFlags.projWin, md)
			if err != nil {
				return err
			}
			rda.WithWindow(*tileWindow)(template)
		}

		// Submit as a batch job.
		resp, err := template.BatchRealize(ctx, rda.Tif)
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(resp)
	},
}

var dgstripMetadataCmd = &cobra.Command{
	Use:   "metadata <catalog-id>",
	Short: "Get metadata describing a realization of a DigitalGlobe strip from RDA",
	Args:  cobra.ExactArgs(1),
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

		// Get the metadata.
		catID := args[0]
		template := rda.NewTemplate(dgstripTemplateID, client, dgstripTemplateOptions(catID)...)
		md, err := template.Metadata()
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(md)
	},
}

func dgstripTemplateOptions(catalogID string) []rda.TemplateOption {
	options := []rda.TemplateOption{
		rda.AddParameter("catalogId", catalogID),
		rda.AddParameter("crs", dgstripFlags.crs.String()),
		rda.AddParameter("bands", dgstripFlags.bt.String()),
		rda.AddParameter("bandSelection", dgstripFlags.bands.String()),
	}

	switch {
	case dgstripFlags.acomp && dgstripFlags.toa:
		options = append(options, rda.AddParameter("correctionType", "Acomp"), rda.AddParameter("fallbackToTOA", "true"))
	case dgstripFlags.acomp:
		options = append(options, rda.AddParameter("correctionType", "Acomp"), rda.AddParameter("fallbackToTOA", "false"))
	case dgstripFlags.toa:
		options = append(options, rda.AddParameter("correctionType", "TOAReflectance"))
	default:
		options = append(options, rda.AddParameter("correctionType", "DN"))
	}

	if dgstripFlags.gsd > 0.0 {
		options = append(options, rda.AddParameter("GSD", fmt.Sprint(dgstripFlags.gsd)))
	}

	if dgstripFlags.dra {
		options = append(options, rda.AddParameter("draType", "HistogramDRA"))
	} else {
		options = append(options, rda.AddParameter("draType", "None"))
	}

	return options
}

var dgstripFlags struct {
	crs   coordRefSys
	acomp bool
	toa   bool
	gsd   float64
	bt    bandType
	bands bandCombo
	dra   bool

	srcWin  sourceWindow
	projWin projectionWindow

	maxconcurr uint64
}

func init() {
	rootCmd.AddCommand(dgstripCmd)
	dgstripCmd.AddCommand(dgstripRealizeCmd)
	dgstripCmd.AddCommand(dgstripMetadataCmd)
	dgstripCmd.AddCommand(dgstripBatchCmd)

	// Control what is fed to the DigitalGlobeStrip template in RDA.
	dgstripCmd.PersistentFlags().Var(&dgstripFlags.crs, "crs", "coordinate reference system to use, either \"UTM\" or \"EPSG:<code>\"")
	dgstripCmd.PersistentFlags().BoolVar(&dgstripFlags.acomp, "acomp", false, "request atmospherically corrected imagery; if --toa is also given, this will default to toa if acomp is not avaiable")
	dgstripCmd.PersistentFlags().BoolVar(&dgstripFlags.toa, "toa", false, "request top of the atmosphere reflectance corrected imagery")
	dgstripCmd.PersistentFlags().Float64Var(&dgstripFlags.gsd, "gsd", 0.0, "ground sample distance; you get native resolution if ommitted")
	dgstripCmd.PersistentFlags().Var(&dgstripFlags.bt, "bandtype", `selected band type, choose "PAN", "MS", "PS", or "SWIR"`)
	dgstripCmd.PersistentFlags().Var(&dgstripFlags.bands, "bands", `selected band combos, choose "ALL", "RGB", or a comma seperated list like "4,2,1"; indexing starts at 0 in the latter case`)
	dgstripCmd.PersistentFlags().BoolVar(&dgstripFlags.dra, "dra", false, "apply a DRA (aka convert to 8 bit in a pretty way)")

	// Local flags specific to realizing tiles.
	dgstripRealizeCmd.Flags().Uint64Var(&dgstripFlags.maxconcurr, "maxconcurrency", 0, "set how many concurrent requests to allow; by default, 4 * num CPUs is used")
	dgstripRealizeCmd.Flags().Var(&dgstripFlags.srcWin, "srcwin", "realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	dgstripRealizeCmd.Flags().Var(&dgstripFlags.projWin, "projwin", "realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")

	// Local flags specific to batch requesting tiles.
	dgstripBatchCmd.Flags().Var(&dgstripFlags.srcWin, "srcwin", "batch realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	dgstripBatchCmd.Flags().Var(&dgstripFlags.projWin, "projwin", "batch realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")
}
