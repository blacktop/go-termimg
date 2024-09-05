/*
Copyright © 2024 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	clihander "github.com/apex/log/handlers/cli"
	"github.com/blacktop/go-termimg"
	"github.com/spf13/cobra"
)

var verbose bool
var clear bool

func init() {
	log.SetHandler(clihander.Default)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&clear, "clear", "c", false, "Clear the image after displaying it")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "imgcat",
	Short: "Display images in your terminal. ",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		timg, err := termimg.Open(args[0])
		if err != nil {
			log.Fatalf("Failed to open image: %v", err)
		}
		defer timg.Close()

		log.Debugf("Image Info: %s", timg.Info())

		out, err := timg.Render()
		if err != nil {
			log.Fatalf("Failed to display image: %v", err)
		}
		fmt.Println(out)

		if clear { // Clear the image after displaying it
			time.Sleep(1 * time.Second)

			out, err = timg.Clear()
			if err != nil {
				log.Fatalf("Failed to clear image: %v", err)
			}
			fmt.Println(out)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
