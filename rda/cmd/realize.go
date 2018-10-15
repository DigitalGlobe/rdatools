package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	srcWin  sourceWindowFlag
	projWin projectionWindowFlag
)

// realizeCmd represents the realize command
var realizeCmd = &cobra.Command{
	Use:   "realize",
	Short: "Materialize the tiles that compose a graph and wrap it in a VRT",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(srcWin, projWin)
		return nil

		// graphID, nodeID := args[0], args[1]
		// config, err := newConfig()
		// if err != nil {
		// 	return err
		// }

		// client, ts, err := newClient(context.TODO(), &config)
		// if err != nil {
		// 	return err
		// }
		// defer writeConfig(&config, ts)

		// md, err := rda.FetchMetadata(graphID, nodeID, client)
		// if err != nil {
		// 	return err
		// }

		// return json.NewEncoder(os.Stdout).Encode(md)

		// log.SetFlags(log.Lshortfile)

		// graphID, nodeID := args[0], args[1]
		// shortGraphID := graphID
		// if len(shortGraphID) > 10 {
		// 	shortGraphID = shortGraphID[0:10]
		// }

		// config, err := newConfig()
		// if err != nil {
		// 	return err
		// }

		// md := Metadata(graphID, nodeID, config)

		// // md.ImageMetadata.NumXTiles = 3
		// // md.ImageMetadata.NumYTiles = 3

		// // Download tiles.
		// r, err := rda.NewRetriever(graphID, nodeID, *md, fmt.Sprintf("%s-%s", shortGraphID, nodeID), config.Token)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// bar := pb.StartNew(md.ImageMetadata.NumXTiles * md.ImageMetadata.NumYTiles)

		// ts := time.Now()
		// tileMap := r.Retrieve(bar.Increment)
		// elapsed := time.Since(ts)
		// bar.FinishPrint("The End!")
		// log.Printf("Tile retrieval took %s", elapsed)

		// // Build VRT.
		// vrt, err := rda.NewVRT(md, tileMap)
		// if err != nil {
		// 	log.Fatal("failed building vrt")
		// }

		// vrtFile := fmt.Sprintf("%s-%s.vrt", shortGraphID, nodeID)
		// f, err := os.Create(vrtFile)
		// if err != nil {
		// 	log.Fatal("failure creating VRT file")
		// }
		// defer f.Close()

		// enc := xml.NewEncoder(f)
		// enc.Indent("  ", "    ")
		// if err := enc.Encode(vrt); err != nil {
		// 	log.Fatalf("error: %v\n", err)
		// }

		// fmt.Printf("VRT available at %s\n", vrtFile)
		// return nil
	},
}

func init() {
	rootCmd.AddCommand(realizeCmd)

	realizeCmd.Flags().Var(&srcWin, "srcwin", "realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	realizeCmd.Flags().Var(&projWin, "projwin", "realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")
}

type sourceWindowFlag struct {
	xOff, yOff, xSize, ySize int
}

func (s *sourceWindowFlag) String() string {
	return "0,0,0,0"
}

func (s *sourceWindowFlag) Set(value string) error {
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

func (s *sourceWindowFlag) Type() string {
	return "int,int,int,int"
}

type projectionWindowFlag struct {
	ulx, uly, lrx, lry float64
}

func (p *projectionWindowFlag) String() string {
	return "0,0,0,0"
}

func (p *projectionWindowFlag) Set(value string) error {
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

func (p *projectionWindowFlag) Type() string {
	return "float,float,float,float"
}
