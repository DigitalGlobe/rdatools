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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const configName = "credentials"

// these are populated by goreleaser when you build a release with that tool.
var (
	version = "head"
	commit  = "head"
	date    = "none"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "rda",
	Long: `A CLI for accessing RDA functionality.

rda can be configured using the 'rda configure' command to store your
GBDX credentials, or by setting the environment variables
'GBDX_USERNAME' and 'GBDX_PASSWORD'.  If you use 'rda configure', you
won't need to bother with your credentials again, as rda handles token
refresh and caching for you.

rda authorization supports "profiles" if you have more than one set of
credentials.  By default, "default" is used if you don't specify a
particual profile via the --profile flag.
`,
	Version: fmt.Sprintf("%v, commit %v, built at %v", version, commit, date),
	// RunE: func(cmd *cobra.Command, args []string) error {
	// 	viper.Debug()
	// 	c, err := newConfig()
	// 	fmt.Printf("%+v\n", c)
	// 	return err
	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("profile", "default", "RDA profile to use")
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))

	viper.BindEnv("gbdx_username")
	viper.BindEnv("gbdx_password")

	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// We map a user defined profile from the cli to the active profile.
	viper.RegisterAlias("ActiveConfig", viper.GetString("profile"))

	// RDA home directory where the config file would be located.
	rdaPath, err := rdaDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed getting path of rda directory, err: %+v\n", err)
		os.Exit(1)
	}

	// Where to find the configuration file, if it exists.
	viper.SetConfigName(configName) // name of rda config file (without extension)
	viper.AddConfigPath(rdaPath)    // adding rda directory as first search path
	viper.ReadInConfig()
}
