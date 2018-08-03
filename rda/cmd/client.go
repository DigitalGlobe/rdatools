package cmd

import (
	"context"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/oauth2"
)

const (
	tokenEndpoint = "https://geobigdata.io/auth/v1/oauth/token"
)

// NewClient returns a rda.Client configured with oauth2 and retry.
func newClient(ctx context.Context, config *Config) (rda.Client, oauth2.TokenSource, error) {
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
