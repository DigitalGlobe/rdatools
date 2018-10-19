package cmd

import (
	"context"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"path"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	srcWin  sourceWindow
	projWin projectionWindow
)

// realizeCmd represents the realize command
var realizeCmd = &cobra.Command{
	Use:   "realize <graph-id> <node-id> <output-vrt>",
	Short: "Materialize the tiles that compose a graph and wrap it in a VRT",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if (projWin != projectionWindow{} && srcWin != sourceWindow{}) {
			return errors.New("--projwin and --srcwin cannot be set at the same time")
		}

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

		md, err := rda.FetchMetadata(graphID, nodeID, client)
		if err != nil {
			return err
		}

		// Convert projWin into a srcWin if we were given one.
		if (projWin != projectionWindow{}) {
			igt, err := md.ImageGeoreferencing.Invert()
			if err != nil {
				return err
			}
			xOff, yOff := igt.Apply(projWin.ulx, projWin.uly)
			srcWin.xOff = int(math.Floor(xOff))
			srcWin.yOff = int(math.Floor(yOff))

			xOffLR, yOffLR := igt.Apply(projWin.lrx, projWin.lry)
			srcWin.xSize = int(math.Ceil(xOffLR - xOff))
			srcWin.ySize = int(math.Ceil(yOffLR - yOff))
		}
		tileWindow, err := md.Subset(srcWin.xOff, srcWin.yOff, srcWin.xSize, srcWin.ySize)
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
		tiles, err := realizer.Realize(context.TODO(), graphID, nodeID, *tileWindow, tileDir, bar.Increment)
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

	realizeCmd.Flags().Var(&srcWin, "srcwin", "realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	realizeCmd.Flags().Var(&projWin, "projwin", "realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")
}

type sourceWindow struct {
	xOff, yOff, xSize, ySize int
}

func (s *sourceWindow) String() string {
	return ""
}

func (s *sourceWindow) Set(value string) error {
	vals := strings.SplitN(value, ",", 4)
	if len(vals) != 4 {
		return fmt.Errorf("expected 4 values, but got %d", len(vals))
	}
	var err error
	if s.xOff, err = strconv.Atoi(vals[0]); err != nil {
		return fmt.Errorf("failed setting xOff = %s, err := %+v", vals[0], err)
	}
	if s.yOff, err = strconv.Atoi(vals[1]); err != nil {
		return fmt.Errorf("failed setting yOff = %s, err := %+v", vals[1], err)
	}
	if s.xSize, err = strconv.Atoi(vals[2]); err != nil {
		return fmt.Errorf("failed setting xSize = %s, err := %+v", vals[2], err)
	}
	if s.ySize, err = strconv.Atoi(vals[3]); err != nil {
		return fmt.Errorf("failed setting ySize = %s, err := %+v", vals[3], err)
	}
	return nil
}

func (s *sourceWindow) Type() string {
	return "int,int,int,int"
}

type projectionWindow struct {
	ulx, uly, lrx, lry float64
}

func (p *projectionWindow) String() string {
	return ""
}

func (p *projectionWindow) Set(value string) error {
	vals := strings.SplitN(value, ",", 4)
	if len(vals) != 4 {
		return fmt.Errorf("expected 4 values, but got %d", len(vals))
	}
	var err error
	if p.ulx, err = strconv.ParseFloat(vals[0], 64); err != nil {
		return fmt.Errorf("failed setting ulx = %s, err := %+v", vals[0], err)
	}
	if p.uly, err = strconv.ParseFloat(vals[1], 64); err != nil {
		return fmt.Errorf("failed setting uly = %s, err := %+v", vals[1], err)
	}
	if p.lrx, err = strconv.ParseFloat(vals[2], 64); err != nil {
		return fmt.Errorf("failed setting lrx = %s, err := %+v", vals[2], err)
	}
	if p.lry, err = strconv.ParseFloat(vals[3], 64); err != nil {
		return fmt.Errorf("failed setting lry = %s, err := %+v", vals[3], err)
	}
	return nil
}

func (p *projectionWindow) Type() string {
	return "float,float,float,float"
}
