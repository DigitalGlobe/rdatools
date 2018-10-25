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
	"net/http"
	"os"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// operatorCmd represents the operator command
var operatorCmd = &cobra.Command{
	Use:   "operator <operator-name>",
	Short: "Return JSON describing the operator(s)",
	Long:  "Return the RDA description of the provided operator.  If the operator is omitted, return this info for all operators.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := newConfig()
		if err != nil {
			return err
		}

		client, ts, err := newClient(context.TODO(), &config)
		if err != nil {
			return err
		}
		defer writeConfig(&config, ts)

		urlPath := rda.OperatorEndpoint
		if len(args) > 0 {
			urlPath = fmt.Sprintf("%s/%s", urlPath, args[0])
		}
		res, err := client.Get(urlPath)
		if err != nil {
			return errors.Wrapf(err, "failure requesting %s", urlPath)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return errors.Errorf("failed fetching operator info from %s, HTTP Status: %s", urlPath, res.Status)
		}

		_, err = io.Copy(os.Stdout, res.Body)
		return errors.Wrap(err, "failed copying response body to stdout")
	},
}

func init() {
	rootCmd.AddCommand(operatorCmd)
}
