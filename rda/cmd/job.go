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
	"log"
	"os"
	"strings"

	"github.com/DigitalGlobe/rdatools/rda/pkg/gbdx"
	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
)

// jobstatusCmd represents the jobstatus command
var jobstatusCmd = &cobra.Command{
	Use:   "jobstatus <job id>",
	Short: "get the status an RDA batch materialization job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobID := args[0]

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

		job, err := rda.FetchBatchStatus(jobID, client)
		return json.NewEncoder(os.Stdout).Encode(job)
	},
}

// jobsdoneCmd represents the jobsdone command
var jobsdoneCmd = &cobra.Command{
	Use:   "jobsdone",
	Short: "returns the list of completed RDA batch materialization job ids",
	Args:  cobra.ExactArgs(0),
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

		sess, s3loc, err := gbdx.NewAWSSession(client)
		if err != nil {
			return err
		}

		svc := s3.New(sess)
		jobIDs := []string{}
		if err := svc.ListObjectsV2PagesWithContext(ctx, &s3.ListObjectsV2Input{
			Bucket:    &s3loc.Bucket,
			Prefix:    aws.String(strings.Join([]string{s3loc.Prefix, "rda/"}, "/")),
			Delimiter: aws.String("/"),
		}, func(p *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, o := range p.CommonPrefixes {
				keys := strings.Split(aws.StringValue(o.Prefix), "/")
				jobIDs = append(jobIDs, keys[len(keys)-2])
			}
			return true
		}); err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(jobIDs)
	},
}

func init() {
	rootCmd.AddCommand(jobstatusCmd)
	rootCmd.AddCommand(jobsdoneCmd)
}
