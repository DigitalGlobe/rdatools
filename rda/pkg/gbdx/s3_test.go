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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

func TestProvider(t *testing.T) {
	resp := `{
  "S3_secret_key": "secret-key",
  "prefix": "prefix",
  "bucket": "bucket",
  "S3_access_key": "access-key",
  "S3_session_token": "session-token"
}`
	vExp := credentials.Value{
		SecretAccessKey: "secret-key",
		AccessKeyID:     "access-key",
		SessionToken:    "session-token",
		ProviderName:    "GBDX",
	}
	cdExp := CustomerDataLocation{
		Bucket: "bucket",
		Prefix: "prefix",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, resp)
	}))
	defer ts.Close()
	s3CredentialsEndpoint = ts.URL

	client := retryablehttp.NewClient()

	provider, err := NewProvider(client)
	if err != nil {
		t.Fatal(err)
	}

	v, err := provider.Retrieve()
	if err != nil {
		t.Fatal(err)
	}
	if v != vExp {
		t.Fatal("credentials.Value not what was expected")
	}
	if provider.CustomerDataLocation != cdExp {
		t.Fatal("CustomerDataLocation not as expected for the provider")
	}
}

func TestNewAWSSession(t *testing.T) {
	resp := `{
  "S3_secret_key": "secret-key",
  "prefix": "prefix",
  "bucket": "bucket",
  "S3_access_key": "access-key",
  "S3_session_token": "session-token"
}`

	exp := struct {
		Value
		CustomerDataLocation
	}{
		Value: Value{SecretAccessKey: "secret-key",
			AccessKeyID:  "access-key",
			SessionToken: "session-token"},
		CustomerDataLocation: CustomerDataLocation{
			Bucket: "bucket",
			Prefix: "prefix",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, resp)
	}))
	defer ts.Close()
	s3CredentialsEndpoint = ts.URL

	client := retryablehttp.NewClient()
	sess, loc, err := NewAWSSession(client)
	if err != nil {
		t.Fatal(err)
	}

	// Creds must be fetched in order for the provider to be invoked.
	sessCreds, err := sess.Config.Credentials.Get()
	if err != nil {
		t.Fatalf("error getting session creds, err: %+v", err)
	}

	if *loc != exp.CustomerDataLocation {
		t.Fatalf("S3 location %#v != %#v", loc, exp.CustomerDataLocation)
	}
	if aws.StringValue(sess.Config.Region) != "us-east-1" {
		t.Fatal("us-east-1 is not the region set for the AWS session")
	}
	if sessCreds.AccessKeyID != exp.AccessKeyID || sessCreds.SecretAccessKey != exp.SecretAccessKey || sessCreds.SessionToken != exp.SessionToken {
		t.Fatalf("session credentials not set as expected")
	}
}

type mockS3 struct {
	s3iface.S3API
	listFunc   func(aws.Context, *s3.ListObjectsV2Input, func(*s3.ListObjectsV2Output, bool) bool, ...request.Option) error
	delObjects func(aws.Context, *s3.DeleteObjectsInput, ...request.Option) (*s3.DeleteObjectsOutput, error)
}

func (m mockS3) ListObjectsV2PagesWithContext(ctx aws.Context, in *s3.ListObjectsV2Input, f func(*s3.ListObjectsV2Output, bool) bool, opts ...request.Option) error {
	return m.listFunc(ctx, in, f, opts...)
}

func (m mockS3) DeleteObjectsWithContext(ctx aws.Context, in *s3.DeleteObjectsInput, opts ...request.Option) (*s3.DeleteObjectsOutput, error) {
	return m.delObjects(ctx, in, opts...)
}

func TestRDABatchJobPrefixes(t *testing.T) {
	exp := []string{"2a2c79d0-acd4-4ea3-a9a4-c144f85708d3", "4840c2f2-b978-4f7c-81a0-dc2988ca4b15", "5e14dff5-dcce-4009-a4c7-9a96e8cdaf3a"}

	m := mockS3{
		listFunc: func(_ aws.Context, _ *s3.ListObjectsV2Input, f func(*s3.ListObjectsV2Output, bool) bool, _ ...request.Option) error {
			f(&s3.ListObjectsV2Output{CommonPrefixes: []*s3.CommonPrefix{&s3.CommonPrefix{Prefix: aws.String("prefix/rda/2a2c79d0-acd4-4ea3-a9a4-c144f85708d3/")}}}, true)
			f(&s3.ListObjectsV2Output{CommonPrefixes: []*s3.CommonPrefix{
				&s3.CommonPrefix{Prefix: aws.String("prefix/rda/4840c2f2-b978-4f7c-81a0-dc2988ca4b15/")},
				&s3.CommonPrefix{Prefix: aws.String("prefix/rda/5e14dff5-dcce-4009-a4c7-9a96e8cdaf3a/")},
			}}, true)
			return nil
		},
	}

	accessor := S3Accessor{
		dataLoc: CustomerDataLocation{},
		svc:     m,
	}

	jobIDs, err := accessor.RDABatchJobPrefixes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(jobIDs, exp) {
		t.Fatalf("%+v != %+v", jobIDs, exp)
	}
}

type mockDownloader struct {
	s3manageriface.DownloaderAPI
	dlFunc func(aws.Context, io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error)
}

func (m mockDownloader) DownloadWithContext(ctx aws.Context, w io.WriterAt, in *s3.GetObjectInput, f ...func(*s3manager.Downloader)) (int64, error) {
	return m.dlFunc(ctx, w, in, f...)
}

func TestDownloadBatchJobArtifacts(t *testing.T) {
	m := mockS3{
		listFunc: func(_ aws.Context, _ *s3.ListObjectsV2Input, f func(*s3.ListObjectsV2Output, bool) bool, _ ...request.Option) error {
			f(&s3.ListObjectsV2Output{Contents: []*s3.Object{
				&s3.Object{Key: aws.String("prefix/rda/jobid/granule_R0C0.tif")},
				&s3.Object{Key: aws.String("prefix/rda/jobid/granule_R0C1.tif")},
			}}, true)
			f(&s3.ListObjectsV2Output{Contents: []*s3.Object{
				&s3.Object{Key: aws.String("prefix/rda/jobid/granule_R1C0.tif")},
				&s3.Object{Key: aws.String("prefix/rda/jobid/granule_R1C1.tif")},
			}}, true)
			return nil
		},
	}

	accessor := S3Accessor{
		dataLoc: CustomerDataLocation{},
		svc:     m,
		downloader: mockDownloader{
			dlFunc: func(aws.Context, io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error) {
				return 0, nil
			},
		},
		progressFunc: func() int { return 0 },
	}

	tmpDir, err := ioutil.TempDir("", "TestDownloadBatchJobArtifacts-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dlCount, dlFunc, err := accessor.DownloadBatchJobArtifacts(context.Background(), tmpDir, "jobid")
	if err != nil {
		t.Fatal(err)
	}
	if dlCount != 4 {
		t.Fatalf("expected 4 objects to download, but got %d", dlCount)
	}

	if err := dlFunc(); err != nil {
		t.Fatal(err)
	}

	// Count files written.
	files, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 4 {
		t.Fatalf("expected 4 objects written to disk, but got %d", len(files))
	}
}
