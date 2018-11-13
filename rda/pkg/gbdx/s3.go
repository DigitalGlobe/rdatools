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

package gbdx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// AWS Access key ID

// AWS Secret Access Key

// AWS Session Token

type awsInformation struct {
	SecretAccessKey string `json:"S3_secret_key"`
	AccessKeyID     string `json:"S3_access_key"`
	SessionToken    string `json:"S3_session_token"`
	CustomerDataLocation
}

// CustomerDataLocation holds the AWS bucket and prefix of where your GBDX data is stored.
type CustomerDataLocation struct {
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix"`
}

func (c CustomerDataLocation) String() string {
	if (c == CustomerDataLocation{}) {
		return ""
	}
	return fmt.Sprintf("s3://%s/%s", c.Bucket, c.Prefix)
}

func (a *awsInformation) credentials() *credentials.Credentials {
	return credentials.NewStaticCredentials(a.AccessKeyID, a.SecretAccessKey, a.SessionToken)
}

// NewAWSSession returns a aws session.Session configured with GBDX
// credentials for accessing your customer data bucket/location.
func NewAWSSession(client *retryablehttp.Client) (*session.Session, *CustomerDataLocation, error) {
	res, err := client.Get(s3CredentialsEndpoint)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failure requesting %s", s3CredentialsEndpoint)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, nil, errors.Errorf("failed getting AWS access info from %s, HTTP Status: %s", s3CredentialsEndpoint, res.Status)
	}

	awsInfo := awsInformation{}
	if err := json.NewDecoder(res.Body).Decode(&awsInfo); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshaling response from GBDX for getting AWS temporary credentials")
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: awsInfo.credentials(),
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed constructing AWS session from GBDX provided AWS credentials")
	}
	return sess, &awsInfo.CustomerDataLocation, nil
}

// S3Accessor handles access to your GBDX S3 locations.
type S3Accessor struct {
	dataLoc      CustomerDataLocation
	svc          s3iface.S3API
	downloader   s3manageriface.DownloaderAPI
	progressFunc func() int
}

// NewS3Accessor returns a configured S3Accessor.
func NewS3Accessor(client *retryablehttp.Client, options ...S3AccessorOption) (*S3Accessor, error) {
	sess, cdl, err := NewAWSSession(client)
	if err != nil {
		return nil, err
	}
	a := &S3Accessor{
		dataLoc:      *cdl,
		svc:          s3.New(sess),
		downloader:   s3manager.NewDownloader(sess),
		progressFunc: func() int { return 0 },
	}
	for _, opt := range options {
		opt(a)
	}
	return a, nil
}

// S3AccessorOption is a type to use for setting options on an S3Accessor.
type S3AccessorOption func(*S3Accessor)

// WithProgressFunc sets a progress function to be called whenever an artifact finishes downloading from S3.
func WithProgressFunc(progressFunc func() int) S3AccessorOption {
	return func(a *S3Accessor) {
		a.progressFunc = progressFunc
	}
}

// RDABatchJobPrefixes returns all the RDA job ids that appear in your
// GBDX customer data bucket under the "rda" prefix.
func (a *S3Accessor) RDABatchJobPrefixes(ctx context.Context) ([]string, error) {
	jobIDs := []string{}
	if err := a.svc.ListObjectsV2PagesWithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:    &a.dataLoc.Bucket,
		Prefix:    aws.String(strings.Join([]string{a.dataLoc.Prefix, "rda/"}, "/")),
		Delimiter: aws.String("/"),
	}, func(p *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, o := range p.CommonPrefixes {
			keys := strings.Split(aws.StringValue(o.Prefix), "/")
			if len(keys) < 2 {
				continue
			}
			jobIDs = append(jobIDs, keys[len(keys)-2])
		}
		return true
	}); err != nil {
		return nil, errors.Wrap(err, "failed listing RDA job ids from S3 location")
	}

	return jobIDs, nil
}

// RDADeleteBatchJobArtifacts deletes all RDA batch job artifacts from
// S3 associated with the given job id, returning the number deleted.
func (a *S3Accessor) RDADeleteBatchJobArtifacts(ctx context.Context, jobID string) (int, error) {
	// List objects under this jobID.
	objects, err := a.listBatchJobArtifacts(ctx, jobID)
	if err != nil {
		return 0, err
	}

	// Delete them in batches of up to 1000 (an S3 api limit).
	for i := 0; i < len(objects); i += 1000 {
		toDel := s3.DeleteObjectsInput{
			Bucket: aws.String(a.dataLoc.Bucket),
			Delete: &s3.Delete{
				Objects: []*s3.ObjectIdentifier{},
			},
		}
		for j := i; j < i+1000 && j < len(objects); j++ {
			toDel.Delete.Objects = append(toDel.Delete.Objects, &s3.ObjectIdentifier{Key: objects[j].Key})
		}

		if _, err := a.svc.DeleteObjectsWithContext(ctx, &toDel); err != nil {
			return 0, errors.Wrapf(err, "failed deleting artifacts associated with RDA job id %s from S3", jobID)
		}
	}
	return len(objects), nil
}

// DownloadBatchJobArtifacts returns the count of objects that will be
// downloaded and a function to run that initiates the download of the
// RDA batch artifacts associated with the given jobID. If the file
// already exists in outDir (taking the same name as in S3), it will
// not be downloaded and won't be counted in the returned count.
//
// We return in this style so that the user can instantiate a progress
// bar if they like; you can provide a function via WithProgressFunc,
// and it will be invokded on every successful download.
func (a *S3Accessor) DownloadBatchJobArtifacts(ctx context.Context, outDir string, jobID string) (int, func() error, error) {
	if err := os.MkdirAll(outDir, 0775); err != nil {
		return 0, nil, err
	}

	possibleDL, err := a.listBatchJobArtifacts(ctx, jobID)
	if err != nil {
		return 0, nil, err
	}

	// Filter out any we've already downloaded.
	toDL := []downloadLocation{}
	for _, obj := range possibleDL {
		_, suffix := path.Split(*obj.Key)
		file := filepath.Join(outDir, suffix)

		if _, err := os.Stat(file); !os.IsNotExist(err) {
			a.progressFunc()
			continue
		}

		toDL = append(toDL, downloadLocation{file: file, object: obj})
	}

	return len(toDL), func() error { return a.downloadArtifacts(ctx, outDir, toDL) }, nil
}

func (a *S3Accessor) listBatchJobArtifacts(ctx context.Context, jobID string) ([]*s3.GetObjectInput, error) {
	objects := []*s3.GetObjectInput{}
	if err := a.svc.ListObjectsV2PagesWithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: &a.dataLoc.Bucket,
		Prefix: aws.String(strings.Join([]string{a.dataLoc.Prefix, "rda", jobID}, "/")),
	}, func(p *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, o := range p.Contents {
			objects = append(objects, &s3.GetObjectInput{Bucket: &a.dataLoc.Bucket, Key: o.Key})
		}
		return true
	}); err != nil {
		return nil, errors.Wrapf(err, "failing listing artifacts associated with RDA batch job %s", jobID)
	}
	return objects, nil
}

type downloadLocation struct {
	file   string
	object *s3.GetObjectInput
}

func (a *S3Accessor) downloadArtifacts(ctx context.Context, outDir string, dlLoc []downloadLocation) error {
	for _, dl := range dlLoc {
		obj, file := dl.object, dl.file

		if err := a.downloadArtifact(ctx, file, obj); err != nil {
			return err
		}
		a.progressFunc()
	}
	return nil
}

func (a *S3Accessor) downloadArtifact(ctx context.Context, file string, obj *s3.GetObjectInput) error {
	fd, err := os.Create(file)
	if err != nil {
		return errors.Wrapf(err, "failed creating file to hold rda output from s3")
	}

	// Delete the file we've created if we didn't download it successfully.
	defer func() {
		if err != nil {
			if nerr := os.Remove(file); nerr != nil {
				err = errors.WithMessagef(err, "failed removing partially downloaded file %s, err: %v", file, nerr)
			}
		}
	}()
	defer fd.Close()

	if _, err = a.downloader.DownloadWithContext(ctx, fd, obj); err != nil {
		return errors.Wrap(err, "failure downloading object from S3")
	}
	return nil
}
