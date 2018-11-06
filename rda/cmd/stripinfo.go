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
	"log"
	"os"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// stripinfoCmd represents the stripinfo command
var stripinfoCmd = &cobra.Command{
	Use:   "stripinfo <catalog-id>",
	Short: "Returns RDA strip level metadata for the given catalog id",

	Long: `RDA strip level metadata details exactly what materials
	the underlying catalog id that is stored in RDA is composed
	of.  This is different than metadata you get back from
	realized graphs and templates in that those are specific to
	the image you can pull out from those nodes.`,
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

		// No zip file, so stream out json.
		if zipfile == "" {
			return errors.Wrap(rda.StripInfo(client, os.Stdout, args[0], false), "failed copying response body to stdout")
		}

		f, err := os.Create(zipfile)
		if err != nil {
			return errors.Wrap(err, "failed creating zip file to write RDA strip information to")
		}
		defer f.Close()

		return errors.Wrap(rda.StripInfo(client, f, args[0], true), "failed writing RDA strip information as zip file")
	},
}

var zipfile string

func init() {
	rootCmd.AddCommand(stripinfoCmd)
	stripinfoCmd.Flags().StringVar(&zipfile, "zipfile", "", "write zipped metadata to the provided filepath")
}
