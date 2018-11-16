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
	"os/signal"
	"syscall"
	"time"

	"github.com/DigitalGlobe/rdatools/rda/pkg/gbdx"
	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "commands (status, download, etc) related to RDA batch materialization job",
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status <job id>*",
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

		// Our HTTP client.
		client, writeConfig, err := newClient(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err := writeConfig(); err != nil {
				log.Printf("on exit, received an error when writing configuration, err: %v", err)
			}
		}()

		// Fetch all the job statuses.
		jobs, err := rda.FetchBatchStatus(ctx, client, args...)
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(jobs)
	},
}

// downloadableCmd represents the downloadable command
var downloadableCmd = &cobra.Command{
	Use:   "downloadable <job id>*",
	Short: "returns the list of RDA batch materialization job ids found in the GBDX customer data bucket",
	Long: `returns the list of RDA batch materialization job ids found in the GBDX customer data bucket 

The returned job ids are available for download, although they may not 
be completed.  You can status them via "rda job status". If you specifiy 
a job id as the second argument, you will instead get a list of all 
objects associated with that job id.`,
	Args: cobra.MaximumNArgs(1),
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

		accessor, err := gbdx.NewS3Accessor(client)
		if err != nil {
			return err
		}

		// Return a listing of all available job ids if no job id was specified.
		if len(args) == 0 {
			jobIDs, err := accessor.RDABatchJobPrefixes(ctx)
			if err != nil {
				return err
			}
			return json.NewEncoder(os.Stdout).Encode(jobIDs)
		}

		// Return a list of all objects associated with this job id.
		paths, err := accessor.RDABatchJobObjects(ctx, args[0])
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(paths)
	},
}

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove S3 artifacts in your GBDX customer data bucket associated with the RDA batch job ID",
	Args:  cobra.ExactArgs(1),
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

		accessor, err := gbdx.NewS3Accessor(client)
		if err != nil {
			return err
		}
		numDel, err := accessor.RDADeleteBatchJobArtifacts(ctx, args[0])
		if err != nil {
			return err
		}
		log.Printf("deleted %d artifacts associated with %s\n", numDel, args[0])
		return nil
	},
}

// download represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download <outdir> <job id>",
	Short: "download RDA batch job artifacts to the output directory",
	Long: `download RDA batch job artifacts to the output directory
 
outdir will be created if it doesn't exist. If you specify the full path 
(vs just the job id) to a file, it will only download that particular file 
rather than the entire job contents.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, jobID := args[0], args[1]

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

		client, writeConfig, err := newClient(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err := writeConfig(); err != nil {
				log.Printf("on exit, received an error when writing configuration, err: %v", err) // TODO, handle more gracefully.
			}
		}()

		accessor, err := gbdx.NewS3Accessor(client)
		if err != nil {
			return err
		}

		numArtifacts, dlFunc, err := accessor.DownloadBatchJobArtifacts(ctx, outDir, jobID)
		if err != nil {
			return err
		}
		if numArtifacts == 0 {
			fmt.Println("no artifacts to download")
			return nil
		}

		bar := pb.StartNew(numArtifacts)
		tStart := time.Now()
		gbdx.WithProgressFunc(bar.Increment)(accessor)
		if err := dlFunc(); err != nil {
			bar.FinishPrint("Failed downloading all artifacts; rerun the command to pick up where you left off.")
			srcErr := errors.Cause(err)
			if aerr, ok := srcErr.(awserr.Error); ok {
				srcErr = aerr.OrigErr()
			}
			if srcErr.Error() != "context canceled" {
				return err
			}
			return nil
		}
		bar.FinishPrint(fmt.Sprintf("S3 download of %d artifacts took %s", numArtifacts, time.Since(tStart)))
		return nil
	},
}

// watch represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch <outdir> <job id>",
	Short: "watch RDA batch job id for completion, greedily downloading artifacts to the output directory as they arrive",
	Long:  `download RDA batch job artifacts to the output directory; ourdir will be created if it doesn't exist`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, jobID := args[0], args[1]

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

		client, writeConfig, err := newClient(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err := writeConfig(); err != nil {
				log.Printf("on exit, received an error when writing configuration, err: %v", err) // TODO, handle more gracefully.
			}
		}()

		// Begin watching the job and downloading granules as they appear.
		status := "processing"
	dlLoop:
		for {
			accessor, err := gbdx.NewS3Accessor(client)
			if err != nil {
				return err
			}

			numDL, dlFunc, err := accessor.DownloadBatchJobArtifacts(ctx, outDir, jobID)
			if err != nil {
				return err
			}

			switch {
			case numDL > 0:
				bar := pb.StartNew(numDL)
				tStart := time.Now()
				gbdx.WithProgressFunc(bar.Increment)(accessor)
				if err := dlFunc(); err != nil {
					bar.FinishPrint("Failed downloading all artifacts; rerun the command to pick up where you left off.")
					srcErr := errors.Cause(err)
					if aerr, ok := srcErr.(awserr.Error); ok {
						srcErr = aerr.OrigErr()
					}
					if srcErr.Error() != "context canceled" {
						return err
					}
					return nil
				}
				bar.FinishPrint(fmt.Sprintf("S3 download of %d artifacts took %s", numDL, time.Since(tStart)))

			case status == "complete":
				// We exit the loop here to ensure there is no more objects to download and the job status is set to complete.
				break dlLoop

			default:
				jobs, err := rda.FetchBatchStatus(ctx, client, jobID)
				if err != nil {
					return err
				}
				if len(jobs) != 1 {
					return errors.Errorf("no job found found for job id %s", jobID)
				}

				switch status = jobs[0].Status.Status; status {
				case "complete":
					continue dlLoop
				case "processing":
				default:
					return errors.Errorf("job id %s has status %s, exiting", jobID, status)
				}

				// If we are still processing but nothing was found to download, sleep for a while before checking again.
				select {
				case <-time.After(10 * time.Second):
				case <-ctx.Done():
					log.Printf("exited before downloading all artifacts; rerun the command to pick up where you left off.")
					return nil
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(jobCmd)
	jobCmd.AddCommand(statusCmd)
	jobCmd.AddCommand(rmCmd)
	jobCmd.AddCommand(downloadableCmd)
	jobCmd.AddCommand(downloadCmd)
	jobCmd.AddCommand(watchCmd)
}
