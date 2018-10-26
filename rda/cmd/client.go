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

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/oauth2"
)

const (
	tokenEndpoint = "https://geobigdata.io/auth/v1/oauth/token"
)

// NewClient returns a rda.Client configured with oauth2 and retry.
func newClient(ctx context.Context, config *Config) (*retryablehttp.Client, oauth2.TokenSource, error) {
	oauth2Conf := &oauth2.Config{
		Endpoint: oauth2.Endpoint{TokenURL: tokenEndpoint},
	}

	// Configure the token source.
	if config.Token == nil {
		var err error
		config.Token, err = oauth2Conf.PasswordCredentialsToken(ctx, config.Username, config.Password)
		if err != nil {
			return nil, nil, err
		}
	}
	tokenSource := oauth2Conf.TokenSource(ctx, config.Token)

	// Configure http retrying.
	client := retryablehttp.NewClient()
	client.HTTPClient = oauth2.NewClient(ctx, tokenSource)
	client.Logger = nil

	return client, tokenSource, nil
}
