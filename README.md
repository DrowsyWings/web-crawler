# Go Web Scraper

A simple web scraper built with Go and Colly to extract product details from an e-commerce site.

## Features
- Scrapes product names, prices, and images
- Saves data to `products.csv`
- Avoids duplicate URLs

## Setup
```sh
go mod init scraper
go mod tidy
go run main.go
```

## Dependencies
- `github.com/gocolly/colly`


