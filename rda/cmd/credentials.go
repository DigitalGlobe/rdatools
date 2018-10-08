package cmd

import (
	"net/http"

	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Config struct {
	Username string        `mapstructure:"gbdx_username" toml:"gbdx_username"`
	Password string        `mapstructure:"gbdx_password" toml:"gbdx_password"`
	Token    *oauth2.Token `mapstructure:"gbdx_token" toml:"gbdx_token"`
}

type Client struct {
	tokenSource oauth2.TokenSource // This guy is kept around so we can cache the token for reuse.
	http.Client
}

func NewConfig() (Config, error) {
	// Load the existing profile, if there is one.
	var profile struct {
		ActiveConfig Config
	}
	if err := viper.Unmarshal(&profile); err != nil {
		return Config{}, err
	}
	return profile.ActiveConfig, nil
}
