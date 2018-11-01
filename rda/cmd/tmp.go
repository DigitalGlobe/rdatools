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
	"log"
	"strings"

	"github.com/DigitalGlobe/rdatools/rda/pkg/gbdx"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
)

// tmpCmd represents the tmp command
var tmpCmd = &cobra.Command{
	Hidden: true,
	Use:    "tmp",
	Short:  "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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

		//urlPath := "https://rda.geobigdata.io/v1/template/DigitalGlobeStrip"
		//urlPath := "https://rda.geobigdata.io/v1/template/materialize/formats"

		// res, err := client.Get(urlPath)
		// if err != nil {
		// 	return errors.Wrapf(err, "failure requesting %s", urlPath)
		// }
		// defer res.Body.Close()
		// if res.StatusCode != http.StatusOK {
		// 	return errors.Errorf("failed fetching operator info from %s, HTTP Status: %s", urlPath, res.Status)
		// }
		//_, err = io.Copy(os.Stdout, res.Body)
		// return errors.Wrap(err, "failed copying response body to stdout")

		sess, s3loc, err := gbdx.NewAWSSession(client)
		if err != nil {
			return err
		}

		svc := s3.New(sess)
		s3Out, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:    &s3loc.Bucket,
			Prefix:    aws.String(strings.Join([]string{s3loc.Prefix, "rda/"}, "/")),
			Delimiter: aws.String("/"),
		})
		if err != nil {
			return err
		}
		fmt.Println(s3Out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tmpCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tmpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tmpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
