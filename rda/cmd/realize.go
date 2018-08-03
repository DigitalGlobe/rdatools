package cmd

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"path"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var graphFlags struct {
	srcWin  sourceWindow
	projWin projectionWindow
}

// realizeCmd represents the realize command
var realizeCmd = &cobra.Command{
	Hidden: true,
	Use:    "realize <graph-id> <node-id> <output-vrt>",
	Short:  "Materialize the tiles that compose a graph and wrap it in a VRT",
	Args:   cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		graphID, nodeID, vrtPath := args[0], args[1], args[2]

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

		tileWindow, err := processSubWindows(&graphFlags.srcWin, &graphFlags.projWin, md)
		if err != nil {
			return err
		}

		// Get the tiles.
		bar := pb.StartNew(tileWindow.NumXTiles * tileWindow.NumYTiles)

		realizer := rda.Realizer{
			Client: client,
		}
		tileDir := vrtPath[:len(vrtPath)-len(path.Ext(vrtPath))]
		tStart := time.Now()
		tiles, err := realizer.RealizeGraph(context.TODO(), graphID, nodeID, *tileWindow, tileDir, bar.Increment)
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

func init() {
	rootCmd.AddCommand(realizeCmd)

	realizeCmd.Flags().Var(&graphFlags.srcWin, "srcwin", "realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	realizeCmd.Flags().Var(&graphFlags.projWin, "projwin", "realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")
}
