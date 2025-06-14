package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"web-crawler/internal/crawler"
	"web-crawler/internal/stats"
	"web-crawler/internal/storage"
	"web-crawler/pkg/models"

	"github.com/boltdb/bolt"
	"github.com/spf13/cobra"
)

var (
	urlFlag    string
	depthFlag  string
	workers    string
	delay      string
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

		db, err := bolt.Open("crawler.db", 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		if err := storage.Init(db); err != nil {
			log.Fatal(err)
		}

		s := stats.NewStats()
		go s.StartReporting()
		defer func() { s.DoneCh <- struct{}{} }()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-quit
			log.Println("Interrupted")
			s.DoneCh <- struct{}{}
			os.Exit(0)
		}()

		config := models.CrawlConfig{
			SeedUrl:            urlFlag,
			Depth:              depthFlag,
			RateLimits:         workers,
			DomainRestrictions: delay,
		}

		c := crawler.NewCrawler(config, db, s)
		c.Start()

		if outputPath != "" {
			results, err := storage.ExportResults(db)
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
	crawlCmd.Flags().StringVar(&depthFlag, "depth", "2", "Maximum crawl depth")
	crawlCmd.Flags().StringVar(&workers, "workers", "4", "Number of workers")
	crawlCmd.Flags().StringVar(&delay, "delay", "0", "Delay between requests")
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
