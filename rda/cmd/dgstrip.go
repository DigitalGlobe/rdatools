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
	"net/url"
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

var dgstripFlags struct {
	crs   coordRefSys
	acomp bool
	toa   bool
	gsd   float64
	bt    bandType
	bands bandCombo
	dra   bool

	srcWin     sourceWindow
	projWin    projectionWindow
	maxconcurr uint64
}

// dgstripCmd represents the dgstrip command
var dgstripCmd = &cobra.Command{
	Use:   "dgstrip <catalog-id> <output-vrt>",
	Short: "Realize tiles of a DigitalGlobe strip from RDA",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get our parameters sorted out.
		catID, vrtPath := args[0], args[1]
		params := map[string]string{"catalogId": catID}
		qp := queryParams(params)

		config, err := newConfig()
		if err != nil {
			return err
		}

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

		// Get the metadata and figure out what tiles we want to pull.
		client, ts, err := newClient(ctx, &config)
		if err != nil {
			return err
		}
		defer writeConfig(&config, ts)

		md, err := rda.TemplateMetadata("DigitalGlobeStrip", client, qp)
		if err != nil {
			return err
		}

		tileWindow, err := processSubWindows(&dgstripFlags.srcWin, &dgstripFlags.projWin, md)
		if err != nil {
			return err
		}

		// Get the tiles.
		bar := pb.StartNew(tileWindow.NumXTiles * tileWindow.NumYTiles)

		realizer := rda.Realizer{
			Client:      client,
			NumParallel: int(dgstripFlags.maxconcurr),
		}
		tileDir := vrtPath[:len(vrtPath)-len(path.Ext(vrtPath))]
		tStart := time.Now()
		tiles, err := realizer.RealizeTemplate(ctx, "DigitalGlobeStrip", qp, *tileWindow, tileDir, bar.Increment)
		if err != nil {
			return err
		}
		bar.FinishPrint(fmt.Sprintf("Tile retrieval took %s", time.Since(tStart)))
		if len(tiles) < 1 {
			return err
		}

		// Build VRT struct and write it to disk.
		vrt, err := rda.NewVRT(md, tiles)
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

var dgstripMetadataCmd = &cobra.Command{
	Use:   "metadata <catalog-id>",
	Short: "Get metadata desribing a realization of a DigitalGlobe strip from RDA",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		params := map[string]string{"catalogId": args[0]}
		qp := queryParams(params)

		config, err := newConfig()
		if err != nil {
			return err
		}

		client, ts, err := newClient(context.TODO(), &config)
		if err != nil {
			return err
		}
		defer writeConfig(&config, ts)

		md, err := rda.TemplateMetadata("DigitalGlobeStrip", client, qp)
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(md)
	},
}

func queryParams(params map[string]string) url.Values {
	qp := make(url.Values)
	for key, val := range params {
		qp.Add(key, val)
	}

	qp.Add("crs", dgstripFlags.crs.String())

	switch {
	case dgstripFlags.acomp && dgstripFlags.toa:
		qp.Add("correctionType", "Acomp")
		qp.Add("fallbackToTOA", "true")
	case dgstripFlags.acomp:
		qp.Add("correctionType", "Acomp")
		qp.Add("fallbackToTOA", "false")
	case dgstripFlags.toa:
		qp.Add("correctionType", "TOAReflectance")
	default:
		qp.Add("correctionType", "DN")
	}

	if dgstripFlags.gsd > 0.0 {
		qp.Add("GSD", fmt.Sprint(dgstripFlags.gsd))
	}

	qp.Add("bands", dgstripFlags.bt.String())

	qp.Add("bandSelection", dgstripFlags.bands.String())

	if dgstripFlags.dra {
		qp.Add("draType", "HistogramDRA")
	} else {
		qp.Add("draType", "None")
	}
	return qp
}

func init() {
	rootCmd.AddCommand(dgstripCmd)
	dgstripCmd.AddCommand(dgstripMetadataCmd)

	// Control what is fed to the DigitalGlobeStrip template in RDA.
	dgstripCmd.PersistentFlags().Var(&dgstripFlags.crs, "crs", "coordinate reference system to use, either \"UTM\" or \"EPSG:<code>\"")
	dgstripCmd.PersistentFlags().BoolVar(&dgstripFlags.acomp, "acomp", false, "request atmospherically corrected imagery; if --toa is also given, this will default to toa if acomp is not avaiable")
	dgstripCmd.PersistentFlags().BoolVar(&dgstripFlags.toa, "toa", false, "request top of the atmosphere reflectance corrected imagery")
	dgstripCmd.PersistentFlags().Float64Var(&dgstripFlags.gsd, "gsd", 0.0, "ground sample distance; you get native resolution if ommitted")
	dgstripCmd.PersistentFlags().Var(&dgstripFlags.bt, "bandtype", `selected band type, choose "PAN", "MS", "PS", or "SWIR"`)
	dgstripCmd.PersistentFlags().Var(&dgstripFlags.bands, "bands", `selected band combos, choose "ALL", "RGB", or a comma seperated list like "4,2,1"; indexing starts at 0 in the latter case`)
	dgstripCmd.PersistentFlags().BoolVar(&dgstripFlags.dra, "dra", false, "apply a DRA (aka convert to 8 bit in a pretty way)")

	// Local flags specific to fetching tiles.
	dgstripCmd.Flags().Uint64Var(&dgstripFlags.maxconcurr, "maxconcurrency", 0, "set how many concurrent requests to allow; by default, 4 * num CPUs is used")
	dgstripCmd.Flags().Var(&dgstripFlags.srcWin, "srcwin", "realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	dgstripCmd.Flags().Var(&dgstripFlags.projWin, "projwin", "realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")
}
