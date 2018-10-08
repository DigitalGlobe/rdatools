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
	"fmt"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rda",
	Short: "A CLI for accessing RDA functionality",
	//Long: `A longer description.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		var config Config
		viper.UnmarshalKey(viper.GetString("profile"), &config)

		fmt.Printf("\n\nconfig at %s is:\n\n%+v\n\n", viper.ConfigFileUsed(), config)

		viper.Debug()
	},
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

	viper.BindEnv("gbdx_username", "GBDX_USERNAME")

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

	// Where to find the configuration file.
	viper.SetConfigName("credentials") // name of rda config file (without extension)
	viper.AddConfigPath(rdaPath)       // adding rda directory as first search path
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
	return rdaPath, os.MkdirAll(rdaPath, 0600)
}
