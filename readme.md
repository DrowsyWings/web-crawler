# Web Crawler CLI

A web crawler written in Go. It uses depth-limited BFS to crawl pages starting from a seed URL and stores the results in BoltDB or Redis.

It supports concurrent crawling using multiple workers. It can also run in distributed mode, where multiple instances share the same crawl job through Redis. Crawl state is stored in Redis, so a crawl can be resumed after restarting an instance.

## Features

- Crawls pages using BFS with a configurable maximum depth
- Uses multiple workers to crawl pages concurrently
- Extracts page titles, metadata, and content
- Extracts links from pages and crawls pages from the same domain
- Tracks visited URLs to avoid crawling the same page multiple times
- Supports in-memory and Redis-based frontiers
- Supports BoltDB and Redis for storing crawl results
- Allows multiple instances to work on the same crawl job
- Resumes crawls from Redis after restarting an instance
- Re-queues tasks if a worker fails before completing them
- Tracks crawled, queued, in-flight, and errored pages
- Exports results to a JSON file

## Installation

Using `go install`:

```bash
go install github.com/DrowsyWings/web-crawler@latest
```

Or clone the repository:

```bash
git clone https://github.com/DrowsyWings/web-crawler
cd web-crawler
go mod tidy
go build .
```

Requires Go 1.26 or later.

## Usage

### Single process

Run the crawler without Redis:

```bash
web-crawler crawl --url https://example.com --depth 2 --workers 4 --delay 500ms --output results.json
```

The crawler uses in-memory state for the frontier and BoltDB for storing results.

### Distributed crawling

To run the crawler in Redis mode:

```bash
web-crawler crawl --url https://example.com --depth 3 --workers 8 --mode redis --redis-addr localhost:6379 --job-id mycrawl
```

Multiple instances can work on the same crawl by using the same `--job-id`. The instances share the URL queue and crawl results through Redis.

If an instance is stopped, its unfinished tasks can be picked up again after their leases expire. Restarting an instance with the same job ID allows it to continue working on the existing crawl.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--url` | Required | Seed URL to start crawling from |
| `--depth` | `2` | Maximum crawl depth |
| `--workers` | `4` | Number of worker goroutines per instance |
| `--delay` | `0s` | Delay between requests, e.g. `500ms` |
| `--timeout` | `10s` | HTTP request timeout |
| `--mode` | `memory` | Backend mode: `memory` or `redis` |
| `--redis-addr` | `localhost:6379` | Redis server address |
| `--job-id` | Derived from seed URL | ID used to share a crawl between instances |
| `--output` | None | Path to the JSON output file |

If `--job-id` is not provided, it is generated from the normalized seed URL. Using the same seed URL will therefore produce the same job ID.

## Output

The crawler exports results as a JSON array. Each page contains its URL, title, metadata, timestamp, and HTTP status.

```json
[
  {
    "url": "https://example.com",
    "title": "Example Domain",
    "description": "Example description from the page meta tag",
    "keywords": "example, domain",
    "timestamp": "2026-10-06T22:33:18+05:30",
    "status": "200 OK"
  }
]
```

The `description` and `keywords` fields are extracted from the page's meta tags and are omitted if they are not available.

## Architecture

The crawler has two main components:

- **Frontier:** Manages the URLs waiting to be crawled and tracks which URLs have already been added. It has in-memory and Redis implementations.
- **Store:** Stores the results of crawled pages. It supports BoltDB and Redis.

In distributed mode, Redis is used to share the frontier and results between instances. It also tracks tasks currently being processed.

The Redis backend uses the following keys for each job:

| Key | Type | Purpose |
|---|---|---|
| `crawl:{job}:queue` | List | URLs waiting to be crawled |
| `crawl:{job}:processing` | Sorted set | Tasks currently being processed |
| `crawl:{job}:seen` | Set | URLs already added to the crawl |
| `crawl:{job}:results` | Hash | Results of crawled pages |

Tasks are claimed atomically and tracked using leases. If a worker fails to finish a task, the task can be added back to the queue after its lease expires. Results are stored by URL, which helps avoid duplicate result entries when a task is retried.

## Running with Docker

Build the image:

```bash
docker build -t web-crawler .
```

Run the crawler:

```bash
docker run --rm web-crawler crawl --url https://example.com --output /dev/stdout
```

To run multiple instances using Docker Compose:

```bash
SEED_URL=https://example.com JOB_ID=demo docker compose up --build --scale crawler=3
```

This starts three crawler instances that share a Redis-backed crawl job.

## Development

Run the tests:

```bash
go test ./...
```

Run the tests with the race detector:

```bash
go test -race ./...
```

The in-memory and Redis frontier implementations use the same conformance tests to check that they behave consistently.