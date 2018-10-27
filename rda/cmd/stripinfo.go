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
	"fmt"
	"io"
	"log"
	"net/http"
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

		urlPath := fmt.Sprintf(rda.StripInfoEndpoint, args[0])
		res, err := client.Get(urlPath)
		if err != nil {
			return errors.Wrapf(err, "failure requesting %s", urlPath)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return rda.ResponseToError(res.Body, fmt.Sprintf("failed fetching operator info from %s, HTTP Status: %s", urlPath, res.Status))
		}

		_, err = io.Copy(os.Stdout, res.Body)
		return errors.Wrap(err, "failed copying response body to stdout")
	},
}

func init() {
	rootCmd.AddCommand(stripinfoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stripinfoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stripinfoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
