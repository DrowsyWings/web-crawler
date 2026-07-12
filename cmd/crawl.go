package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DrowsyWings/web-crawler/internal/crawler"
	"github.com/DrowsyWings/web-crawler/internal/frontier"
	"github.com/DrowsyWings/web-crawler/internal/stats"
	"github.com/DrowsyWings/web-crawler/internal/store"
	"github.com/DrowsyWings/web-crawler/pkg/models"

	"github.com/spf13/cobra"
)

var (
	urlFlag    string
	depthFlag  int
	workers    int
	delay      time.Duration
	timeout    time.Duration
	outputPath string
)

// crawlCmd represents the crawl command
var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		if urlFlag == "" {
			log.Fatal("--url is required")
		}

		resultStore, err := store.OpenBolt("crawler.db")
		if err != nil {
			log.Fatal(err)
		}
		defer resultStore.Close()

		stats := stats.NewStats()

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		config := models.Config{
			SeedURL:  urlFlag,
			MaxDepth: depthFlag,
			Workers:  workers,
			Delay:    delay,
			Timeout:  timeout,
		}

		httpClient := &http.Client{Timeout: config.Timeout}
		f := frontier.NewMemory(1000)
		c := crawler.NewCrawler(config, f, resultStore, stats, httpClient)
		c.Start(ctx)

		if outputPath != "" {
			results, err := resultStore.ExportResults(ctx)
			if err != nil {
				log.Printf("Failed to export results: %v\n", err)
				return
			}

			f, err := os.Create(outputPath)
			if err != nil {
				log.Printf("Failed to create output file: %v\n", err)
				return
			}
			defer f.Close()

			enc := json.NewEncoder(f)
			enc.SetIndent("", "  ")
			if err := enc.Encode(results); err != nil {
				log.Printf("Failed to encode JSON: %v\n", err)
			} else {
				fmt.Println("Results exported to", outputPath)
			}
		}
	},
}

func init() {
	crawlCmd.Flags().StringVar(&urlFlag, "url", "", "Seed URL")
	crawlCmd.Flags().IntVar(&depthFlag, "depth", 2, "Maximum crawl depth")
	crawlCmd.Flags().IntVar(&workers, "workers", 4, "Number of workers")
	crawlCmd.Flags().DurationVar(&delay, "delay", 0, "Delay between requests (e.g. 500ms, 1s)")
	crawlCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Per-request HTTP timeout")
	crawlCmd.Flags().StringVar(&outputPath, "output", "", "Path to JSON file")

	rootCmd.AddCommand(crawlCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// crawlCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// crawlCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
