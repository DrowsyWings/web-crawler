/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "web-crawler",
	Short: "A concurrent web crawler",
	Long: `web-crawler is a concurrent, depth-limited web crawler.

Starting from a seed URL, it fetches pages with a pool of workers, extracts
same-domain links, and stores the results. Use the "crawl" subcommand to run
a crawl.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}


