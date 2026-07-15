package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/DrowsyWings/web-crawler/internal/normalize"
	"github.com/DrowsyWings/web-crawler/internal/stats"
	"github.com/DrowsyWings/web-crawler/internal/store"
	"github.com/DrowsyWings/web-crawler/pkg/models"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var (
	urlFlag    string
	depthFlag  int
	workers    int
	delay      time.Duration
	timeout    time.Duration
	outputPath string
	mode       string
	redisAddr  string
	jobID      string
)

func deriveJobID(seed string) string {
	sum := sha256.Sum256([]byte(normalize.Normalize(seed)))
	return hex.EncodeToString(sum[:])[:12]
}

// crawlCmd represents the crawl command
var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "Crawl a site from a seed URL",
	Run: func(cmd *cobra.Command, args []string) {
		if urlFlag == "" {
			log.Fatal("--url is required")
		}

		stats := stats.NewStats()

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		config := models.Config{
			SeedURL:  urlFlag,
			MaxDepth: depthFlag,
			Workers:  workers,
			Delay:    delay,
			Timeout:  timeout,
			JobID:    jobID,
		}
		if config.JobID == "" {
			config.JobID = deriveJobID(config.SeedURL)
		}

		var f frontier.Frontier
		var resultStore store.Store
		var cleanup func()

		switch mode {
		case "memory":
			boltStore, err := store.OpenBolt("crawler.db")
			if err != nil {
				log.Fatal(err)
			}
			f = frontier.NewMemory(1000)
			resultStore = boltStore
			cleanup = func() { boltStore.Close() }
		case "redis":
			// One client shared by frontier and store; cmd owns closing it.
			client := redis.NewClient(&redis.Options{Addr: redisAddr})
			rf := frontier.NewRedis(client, config.JobID, 3*config.Timeout)
			go rf.StartReaper(ctx, 2*time.Second)
			f = rf
			resultStore = store.NewRedis(client, config.JobID)
			cleanup = func() { client.Close() }
			log.Printf("redis mode: job %s @ %s", config.JobID, redisAddr)
		default:
			log.Fatalf("invalid --mode %q (want memory or redis)", mode)
		}
		defer cleanup()

		httpClient := &http.Client{Timeout: config.Timeout}
		c := crawler.NewCrawler(config, f, resultStore, stats, httpClient)
		c.Start(ctx)

		if outputPath != "" {
			results, err := resultStore.ExportResults(ctx)
			if err != nil {
				log.Printf("Failed to export results: %v\n", err)
				return
			}

			out, err := os.Create(outputPath)
			if err != nil {
				log.Printf("Failed to create output file: %v\n", err)
				return
			}
			defer out.Close()

			enc := json.NewEncoder(out)
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
	crawlCmd.Flags().StringVar(&mode, "mode", "memory", "Backend: memory or redis")
	crawlCmd.Flags().StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis address (redis mode)")
	crawlCmd.Flags().StringVar(&jobID, "job-id", "", "Shared job id for cooperating/resumable crawls (default: derived from seed URL)")

	rootCmd.AddCommand(crawlCmd)
}
