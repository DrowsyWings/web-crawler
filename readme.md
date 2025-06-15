# Web Crawler CLI

A basic  web crawler written in Go. Uses  bfs(limited depth) for crawling from a seed URL. Results are stored in BoltDB

## Features

- Extract page titles and metadata
- Tracks the visited url with hashmap
- Stats for crawled, queued, and errored pages

## Installation

```bash
git clone https://github.com/DrowsyWings/web-crawler
cd web-crawler
go mod tidy
````

## Usage

```bash
go run main.go crawl --url https://example.com --depth 2 --workers 4 --delay 500ms --output results.json
```


## Output Example

The JSON output file contains a list of pages like:

```json
[
  {
    "url": "https://example.com",
    "title": "Example Domain",
    "meta": {
      "description": "...",
      "keywords": "..."
    },
    "content": "...",
    "timestamp": "2025-06-15T12:00:00Z",
    "status": "200 OK"
  }
]
```

---
