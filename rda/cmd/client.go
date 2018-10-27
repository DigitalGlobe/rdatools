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

// newClient returns a rda.Client configured with oauth2 and retry.
// Be sure to defer the returned function when a successful call is
// returned to enable updating the token.
func newClient(ctx context.Context) (*retryablehttp.Client, func() error, error) {
	ts, updateConfig, err := newTokenSource(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Configure http retrying.
	client := retryablehttp.NewClient()
	client.HTTPClient = oauth2.NewClient(ctx, ts)
	client.Logger = nil

	return client, updateConfig, nil
}

// newTokenSource returns a configured oauth2 token source and a
// function that when invoked, will update the rda configuration file
// with a new token.
func newTokenSource(ctx context.Context) (oauth2.TokenSource, func() error, error) {
	config, err := newConfig()
	if err != nil {
		return nil, nil, err
	}

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
	ts := oauth2Conf.TokenSource(ctx, config.Token)
	updateConfig := func() error { return writeConfig(&config, ts) }

	return ts, updateConfig, nil
}
