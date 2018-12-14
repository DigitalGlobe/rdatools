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
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "RDA template functionality",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("template called")
	// },
}

var templateDescribeCmd = &cobra.Command{
	Use:   "describe <template id>",
	Short: "describe returns a description of the RDA template",
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

		template := rda.NewTemplate(args[0], client)
		g, err := template.Describe()
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(&g)
	},
}

var templateUploadCmd = &cobra.Command{
	Use:   "upload <template path>",
	Short: "upload uploads a RDA template to the RDA API, returning a template id for it",
	Long: `upload uploads a RDA template to the RDA API, returning a template id for it

Edge ID fields are not required, and will be overwritten. 

You can specifiy a "-" as the path and it will read the template from an input pipe`,
	Args: cobra.ExactArgs(1),
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

		var r io.Reader
		switch file := args[0]; file {
		case "-":
			r = os.Stdin
		default:
			f, err := os.Open(file)
			if err != nil {
				return errors.Wrap(err, "couldn't open template file for uploading")
			}
			defer f.Close()
			r = f
		}

		// We parse the graph in part to figure out if its valid rather than just passing it through.
		g, err := rda.NewGraphFromAPI(r)
		if err != nil {
			return err
		}
		template := rda.NewTemplate(args[0], client)
		id, err := template.Upload(g)
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(struct {
			ID string `json:"id"`
		}{ID: id})
	},
}

var templateMetadataCmd = &cobra.Command{
	Use:   "metadata <template id>",
	Short: "fetch RDA metadata for the given template",
	Long: `fetch RDA metadata for the given template

Use the flag "--kv" to specify the key and value as comma seperated
arguments.  The value will be substituted into the template where ever
the corresponding key is present.  Repeat the flag to substitute
multiple values.`,
	Args: cobra.ExactArgs(1),
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
		templateID := args[0]

		// Deal with the flags.
		var params []rda.TemplateOption
		for _, kv := range templateFlags.keyvals {
			s := strings.SplitN(kv, ",", 2)
			if len(s) != 2 {
				return errors.Errorf("--kv = %q is not of the form \"key,value\"", kv)
			}
			params = append(params, rda.AddParameter(strings.TrimSpace(s[0]), strings.TrimSpace(s[1])))
		}
		if templateFlags.nodeID != "" {
			params = append(params, rda.AddParameter("nodeId", templateFlags.nodeID))
		}

		// Get the metadata.
		template := rda.NewTemplate(templateID, client, params...)
		md, err := template.Metadata()
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(md)
	},
}

// templateRealizeCmd represents the template command
var templateRealizeCmd = &cobra.Command{
	Use:   "realize <template-id> <output-vrt>",
	Short: "Realize tiles out of a node in an RDA template",
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

		// Parse the flags.
		var params []rda.TemplateOption
		for _, kv := range templateFlags.keyvals {
			s := strings.SplitN(kv, ",", 2)
			if len(s) != 2 {
				return errors.Errorf("--kv = %q is not of the form \"key,value\"", kv)
			}
			params = append(params, rda.AddParameter(strings.TrimSpace(s[0]), strings.TrimSpace(s[1])))
		}
		if templateFlags.nodeID != "" {
			params = append(params, rda.AddParameter("nodeId", templateFlags.nodeID))
		}

		// Get the metadata and figure out what RDA tiles need to be downloaded.
		templateID, vrtPath := args[0], args[1]
		template := rda.NewTemplate(templateID, client, params...)
		md, err := template.Metadata()
		if err != nil {
			return err
		}

		tileWindow, err := processSubWindows(&templateFlags.srcWin, &templateFlags.projWin, md)
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

// templateBatchCmd represents the template command
var templateBatchCmd = &cobra.Command{
	Use:   "batch <template-id>",
	Short: "Realize images of a RDA template via RDA batch materialization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Setup our context to handle cancellation and listen for signals.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

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

		// Parse the flags.
		var params []rda.TemplateOption
		for _, kv := range templateFlags.keyvals {
			s := strings.SplitN(kv, ",", 2)
			if len(s) != 2 {
				return errors.Errorf("--kv = %q is not of the form \"key,value\"", kv)
			}
			params = append(params, rda.AddParameter(strings.TrimSpace(s[0]), strings.TrimSpace(s[1])))
		}
		if templateFlags.nodeID != "" {
			params = append(params, rda.AddParameter("nodeId", templateFlags.nodeID))
		}

		// Get the metadata and figure out what RDA tiles need to be downloaded.
		templateID := args[0]
		template := rda.NewTemplate(templateID, client, params...)
		md, err := template.Metadata()
		if err != nil {
			return err
		}

		if md.ImageGeoreferencing.SpatialReferenceSystemCode == "" {
			return errors.New("rda batch materialization requires georeferenced imagery, but we found no EPSG code")
		}

		// mapping to RDA tiles.
		if (templateFlags.projWin != projectionWindow{} || templateFlags.srcWin != sourceWindow{}) {
			md, err := template.Metadata()
			if err != nil {
				return err
			}
			tileWindow, err := processSubWindows(&templateFlags.srcWin, &templateFlags.projWin, md)
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

var templateFlags struct {
	keyvals []string

	nodeID string

	srcWin  sourceWindow
	projWin projectionWindow

	maxconcurr uint64
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateDescribeCmd)
	templateCmd.AddCommand(templateUploadCmd)
	templateCmd.AddCommand(templateMetadataCmd)
	templateCmd.AddCommand(templateRealizeCmd)
	templateCmd.AddCommand(templateBatchCmd)

	// Local flags specific to getting template metadata.
	templateMetadataCmd.Flags().StringArrayVar(&templateFlags.keyvals, "kv", []string{}, "key/value pairs (comma seperated) for template subsitution")
	templateMetadataCmd.Flags().StringVar(&templateFlags.nodeID, "node", "", "node id to evaluate; if absent the default node is evaluated")

	// Local flags specific to getting template tile realization.
	templateRealizeCmd.Flags().StringArrayVar(&templateFlags.keyvals, "kv", []string{}, "key/value pairs (comma seperated) for template subsitution")
	templateRealizeCmd.Flags().StringVar(&templateFlags.nodeID, "node", "", "node id to evaluate; if absent the default node is evaluated")
	templateRealizeCmd.Flags().Uint64Var(&templateFlags.maxconcurr, "maxconcurrency", 0, "set how many concurrent requests to allow; by default, 4 * num CPUs is used")
	templateRealizeCmd.Flags().Var(&templateFlags.srcWin, "srcwin", "realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	templateRealizeCmd.Flags().Var(&templateFlags.projWin, "projwin", "realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")

	// Local flags specific to RDA template batch realization.
	templateBatchCmd.Flags().StringArrayVar(&templateFlags.keyvals, "kv", []string{}, "key/value pairs (comma seperated) for template subsitution")
	templateBatchCmd.Flags().StringVar(&templateFlags.nodeID, "node", "", "node id to evaluate; if absent the default node is evaluated")
	templateBatchCmd.Flags().Var(&templateFlags.srcWin, "srcwin", "batch realize a subwindow in pixel space, specified via comma seperated integers xoff,yoff,xsize,ysize")
	templateBatchCmd.Flags().Var(&templateFlags.projWin, "projwin", "batch realize a subwindow in projected space, specified via comma seperated floats ulx,uly,lrx,lry")
}
