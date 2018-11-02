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
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

func TestNewAWSSession(t *testing.T) {
	resp := `{
  "S3_secret_key": "secret-key",
  "prefix": "prefix",
  "bucket": "bucket",
  "S3_access_key": "access-key",
  "S3_session_token": "session-token"
}`

	exp := awsInformation{
		SecretAccessKey: "secret-key",
		AccessKeyID:     "access-key",
		SessionToken:    "session-token",
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

	if *loc != exp.CustomerDataLocation {
		t.Fatalf("S3 location %+v != %+v", loc, exp.CustomerDataLocation)
	}
	if aws.StringValue(sess.Config.Region) != "us-east-1" {
		t.Fatal("us-east-1 is not the region set for the AWS session")
	}
	sessCreds, err := sess.Config.Credentials.Get()
	if err != nil {
		t.Fatalf("error getting session creds, err: %+v", err)
	}
	if sessCreds.AccessKeyID != exp.AccessKeyID || sessCreds.SecretAccessKey != exp.SecretAccessKey || sessCreds.SessionToken != exp.SessionToken {
		t.Fatalf("session credentials not set as expected")
	}
}

type mockS3 struct {
	s3iface.S3API
	prefixes []*s3.CommonPrefix
}

func (m mockS3) ListObjectsV2PagesWithContext(_ aws.Context, _ *s3.ListObjectsV2Input, f func(*s3.ListObjectsV2Output, bool) bool, _ ...request.Option) error {
	f(&s3.ListObjectsV2Output{CommonPrefixes: m.prefixes[0:1]}, true)
	f(&s3.ListObjectsV2Output{CommonPrefixes: m.prefixes[1:]}, true)
	return nil
}

func TestS3Accessor(t *testing.T) {
	exp := []string{"2a2c79d0-acd4-4ea3-a9a4-c144f85708d3", "4840c2f2-b978-4f7c-81a0-dc2988ca4b15", "5e14dff5-dcce-4009-a4c7-9a96e8cdaf3a"}

	m := mockS3{prefixes: []*s3.CommonPrefix{
		&s3.CommonPrefix{Prefix: aws.String("prefix/rda/2a2c79d0-acd4-4ea3-a9a4-c144f85708d3/")},
		&s3.CommonPrefix{Prefix: aws.String("prefix/rda/4840c2f2-b978-4f7c-81a0-dc2988ca4b15/")},
		&s3.CommonPrefix{Prefix: aws.String("prefix/rda/5e14dff5-dcce-4009-a4c7-9a96e8cdaf3a/")},
	}}

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
