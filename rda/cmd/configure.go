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

	"github.com/spf13/cobra"
)

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

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure RDA access, e.g. store your creds in ~/.rda.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load the existing config, if there is one.
		config, err := NewConfigFromRDADir()
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
		return nil
	},
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
