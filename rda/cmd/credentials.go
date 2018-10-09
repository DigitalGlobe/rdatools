// Copyright © 2018 Patrick Young <patrick.mckendree.young@gmail.com, patrick.young@digitalglobe.com>
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
	"errors"
	"net/http"

	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// Config holds the authorization info needed to access RDA.
type Config struct {
	Username string        `mapstructure:"gbdx_username" toml:"gbdx_username"`
	Password string        `mapstructure:"gbdx_password" toml:"gbdx_password"`
	Token    *oauth2.Token `mapstructure:"gbdx_token" toml:"gbdx_token"`
}

type Client struct {
	tokenSource oauth2.TokenSource // This guy is kept around so we can cache the token for reuse.
	http.Client
}

// NewConfig returns a Config configured by pulling in credentials via viper.
func NewConfig() (Config, error) {
	var config Config
	if err := viper.UnmarshalKey(viper.GetString("profile"), &config); err != nil {
		return Config{}, err
	}
	if viper.IsSet("gbdx_username") && viper.IsSet("gbdx_password") {
		config.Username = viper.GetString("gbdx_username")
		config.Password = viper.GetString("gbdx_password")
		config.Token = nil
	}

	// We expect these to have been set at this point, otherwise the config will be unusable.
	if config.Username == "" {
		return Config{}, errors.New("no username found to use for authorization")
	}
	if config.Password == "" {
		return Config{}, errors.New("no password found to use for authorization")
	}

	return config, nil
}

// NewConfig returns a Config configured by pulling in credentials via viper.
func NewConfigFromRDADir() (Config, error) {
	var config Config
	if _, err := ensureRDADir(); err != nil {
		return Config{}, err
	}
	if err := viper.UnmarshalKey(viper.GetString("profile"), &config); err != nil {
		return Config{}, err
	}
	return config, nil
}
