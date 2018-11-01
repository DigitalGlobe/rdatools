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
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/DigitalGlobe/rdatools/rda/pkg/gbdx"
	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
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
				log.Printf("on exit, received an error when writing configuration, err: %v", err) // TODO, handle more gracefully.
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

// jobdownload represents the jobdownload command
var jobdownloadCmd = &cobra.Command{
	Use:   "jobdownload <outdir> <job id>",
	Short: "download RDA batch job artifacts to the output directory",
	Long:  `download RDA batch job artifacts to the output directory; ourdir will be created if it doesn't exist`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, jobID := args[0], args[1]

		ctx := context.Background()
		client, writeConfig, err := newClient(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err := writeConfig(); err != nil {
				log.Printf("on exit, received an error when writing configuration, err: %v", err) // TODO, handle more gracefully.
			}
		}()

		job, err := rda.FetchBatchStatus(jobID, client)
		if err != nil {
			return err
		}
		if job.Status.Status != "complete" {
			return errors.Errorf("cannot download a job that isn't complete, job status is %q", job.Status.Status)
		}

		sess, s3loc, err := gbdx.NewAWSSession(client)
		if err != nil {
			return err
		}
		objects, err := getRDAJobObjects(ctx, sess, s3loc, jobID)
		if err != nil {
			return errors.Wrapf(err, "failed listing obects for download related to RDA job id %q", jobID)
		}

		if err := os.MkdirAll(outDir, 0775); err != nil {
			return err
		}

		downloader := s3manager.NewDownloader(sess)
		for _, objIn := range objects {
			_, suffix := path.Split(*objIn.Key)
			file := filepath.Join(outDir, suffix)

			fd, err := os.Create(file)
			if err != nil {
				return errors.Wrapf(err, "failed creatingf file to hold rda output from s3")
			}

			err = func() error {
				defer fd.Close()
				log.Printf("downloading %s to %s\n", fmt.Sprintf("s3://%s/%s", *objIn.Bucket, *objIn.Key), file)
				_, err := downloader.DownloadWithContext(ctx, fd, objIn)
				return err
			}()
			if err != nil {
				return errors.Wrapf(err, "failed downloading rda output from s3")
			}
		}

		return nil
	},
}

func getRDAJobObjects(ctx context.Context, sess *session.Session, s3loc *gbdx.CustomerDataLocation, jobID string) ([]*s3.GetObjectInput, error) {
	svc := s3.New(sess)
	objects := []*s3.GetObjectInput{}
	if err := svc.ListObjectsV2PagesWithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: &s3loc.Bucket,
		Prefix: aws.String(strings.Join([]string{s3loc.Prefix, "rda", jobID}, "/")),
	}, func(p *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, o := range p.Contents {
			objects = append(objects, &s3.GetObjectInput{Bucket: &s3loc.Bucket, Key: o.Key})
		}
		return true
	}); err != nil {
		return nil, err
	}
	return objects, nil
}

func init() {
	rootCmd.AddCommand(jobstatusCmd)
	rootCmd.AddCommand(jobsdoneCmd)
	rootCmd.AddCommand(jobdownloadCmd)
}
