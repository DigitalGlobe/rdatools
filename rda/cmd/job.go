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
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/DigitalGlobe/rdatools/rda/pkg/gbdx"
	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
)

// jobstatusCmd represents the jobstatus command
var jobstatusCmd = &cobra.Command{
	Use:   "jobstatus <job id>*",
	Short: "get the status an RDA batch materialization job(s)",
	Long:  `note that you can list job ids as arguments on the command line, or pipe them in from another source`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read job ids from stdin (line seperated) if given no arguments.
		if len(args) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				args = append(args, scanner.Text())
			}
		}

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

		numWorkers := 8
		if len(args) < numWorkers {
			numWorkers = len(args)
		}
		jobIDsIn := make(chan string)
		jobsOut := make(chan *rda.BatchResponse)

		// Send the job ids along the channel to be processed.
		go func(jobIDsIn chan<- string) {
			defer close(jobIDsIn)
			for _, job := range args {
				jobIDsIn <- job
			}
		}(jobIDsIn)

		// Start the workers that are processing the job ids.
		wg := sync.WaitGroup{}
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(jobIDsIn <-chan string, jobsOut chan<- *rda.BatchResponse) {
				defer wg.Done()
				for jobID := range jobIDsIn {
					job, err := rda.FetchBatchStatus(jobID, client)
					if err != nil {
						log.Printf("error statusing job id %s, err: %v", jobID, err)
						continue
					}
					jobsOut <- job
				}
			}(jobIDsIn, jobsOut)
		}

		// When the workers wrap up, close the output channel.
		go func() {
			wg.Wait()
			close(jobsOut)
		}()

		// Read all the responses from the processed job ids.
		jobs := []*rda.BatchResponse{}
		for job := range jobsOut {
			jobs = append(jobs, job)
		}

		return json.NewEncoder(os.Stdout).Encode(jobs)
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
