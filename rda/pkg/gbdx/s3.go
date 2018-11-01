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
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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
	res, err := client.Get(S3CredentialsEndpoint)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failure requesting %s", S3CredentialsEndpoint)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, nil, errors.Errorf("failed getting AWS access info from %s, HTTP Status: %s", S3CredentialsEndpoint, res.Status)
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
