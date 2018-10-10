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
	"fmt"
	"os"
	"path"

	"path/filepath"

	"github.com/BurntSushi/toml"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// Config holds the authorization info needed to access RDA.
type Config struct {
	Username string        `mapstructure:"gbdx_username" toml:"gbdx_username"`
	Password string        `mapstructure:"gbdx_password" toml:"gbdx_password"`
	Token    *oauth2.Token `mapstructure:"gbdx_token" toml:"gbdx_token,omitempty"`
}

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure RDA access, e.g. store your creds in ~/.rda.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load the existing config, if there is one.
		config, err := newConfigFromRDADir()
		if err != nil {
			return err
		}

		// Get the configuration overrides from the user via the command line.
		var configVars = []struct {
			prompt   string
			val      *string
			isSecret bool
		}{
			{"GBDX User Name", &config.Username, false},
			{"GBDX Password", &config.Password, true},
		}
		for _, configVar := range configVars {
			// Pretty print the prompt for this variable.
			fmt.Printf(configVar.prompt)
			if val := *configVar.val; len(val) > 0 {
				if configVar.isSecret {
					fmt.Printf(" [%s]", secretString(val[max(0, len(val)-10):]))
				} else {
					fmt.Printf(" [%s]", val)
				}
			}
			fmt.Printf(": ")

			// Get user input for this value.
			var s string
			if n, err := fmt.Scanln(&s); err != nil && n > 0 {
				// Gobble up remaining tokens if any.
				for n, err := fmt.Scanln(&s); err != nil && n > 0; {
				}
				return fmt.Errorf("your input is bogus: %v", err)
			}
			if len(s) > 0 {
				*configVar.val = s
			}
		}
		return writeConfig(&config)
	},
}

// newConfig returns a Config configured by pulling in credentials via
// viper, overriding GBDX username and passwords if they were given on
// the command line.
func newConfig() (Config, error) {
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

// newConfigFromRDADir returns a Config configured by pulling in credentials from the configuration file.
func newConfigFromRDADir() (Config, error) {
	var config Config
	if err := viper.UnmarshalKey(viper.GetString("profile"), &config); err != nil {
		return Config{}, err
	}
	return config, nil
}

// cacheToken updates an existing configuration file with the
// provided one.  Note that we only update the profile as stored in
// viper.
func writeConfig(config *Config) error {
	// Need the RDA dir around to write the config to.
	rdaDir, err := ensureRDADir()
	if err != nil {
		return err
	}

	// Read in configuration file if it exists.  Note this may contain multiple profiles.
	profilesOut := make(map[string]Config)
	confFile := viper.ConfigFileUsed()
	if confFile == "" {
		confFile = filepath.Join(rdaDir, configName+".toml")
	}

	_, err = toml.DecodeFile(confFile, &profilesOut)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to parse the configurtion file: %v", err)
	}

	// Update this profile and write it to the credentials file.
	profilesOut[viper.GetString("profile")] = *config
	file, err := os.Create(confFile)
	if err != nil {
		return fmt.Errorf("failed to write updated configuration to disk: %v", err)
	}
	defer file.Close()
	return toml.NewEncoder(file).Encode(profilesOut)
}

// rdaDir returns the location of where we store the RDA configuration directory.
func rdaDir() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	rdaPath := path.Join(home, ".rda")
	return rdaPath, nil
}

// ensureRDADir will create the RDA directory if it doesn't already exist.
func ensureRDADir() (string, error) {
	rdaPath, err := rdaDir()
	if err != nil {
		return "", err
	}
	return rdaPath, os.MkdirAll(rdaPath, 0700)
}

type secretString string

// String returns secretString types as a string with hidden entries.
func (s secretString) String() (str string) {
	for i, c := range s {
		if i > 3 && len(s)-i < 5 {
			str += string(c)
		} else {
			str += "*"
		}
	}
	return
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
