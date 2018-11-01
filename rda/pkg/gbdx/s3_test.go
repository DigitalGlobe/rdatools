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
	"encoding/json"
	"testing"
)

var gbdxS3CredResp = `{
  "S3_secret_key": "secret-key",
  "prefix": "prefix",
  "bucket": "bucket",
  "S3_access_key": "access-key",
  "S3_session_token": "session-token"
}`

func TestS3CredUnmarshal(t *testing.T) {
	awsInfo := awsInformation{}
	if err := json.Unmarshal([]byte(gbdxS3CredResp), &awsInfo); err != nil {
		t.Fatal(err)
	}
	expected := awsInformation{
		SecretAccessKey: "secret-key",
		AccessKeyID:     "access-key",
		SessionToken:    "session-token",
		CustomerDataLocation: CustomerDataLocation{
			Bucket: "bucket",
			Prefix: "prefix",
		},
	}

	if awsInfo != expected {
		t.Fatalf("%+v != %+v", awsInfo, expected)
	}

}
