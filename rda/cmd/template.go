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
	"io"
	"log"
	"os"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
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

You can specifiy a "-" as the path and it will read the template from an input pipe`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// ctx := context.Background()
		// client, writeConfig, err := newClient(ctx)
		// if err != nil {
		// 	return err
		// }
		// defer func() {
		// 	if err := writeConfig(); err != nil {
		// 		log.Printf("on exit, received an error when writing configuration, err: %v", err)
		// 	}
		// }()

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

		g, err := rda.NewGraphFromAPI(r)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(&g)
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateDescribeCmd)
	templateCmd.AddCommand(templateUploadCmd)
}
