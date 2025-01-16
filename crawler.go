package main

import (
	"fmt"
	"encoding/csv"
	"os"

	"github.com/gocolly/colly"
)

var visitedurls = make(map[string]bool)

type Product struct {
	Name     string
	Price    string
	ImageURL string
}

var products []Product

func main() {
	seedurl := "https://www.scrapingcourse.com/ecommerce/"

	crawl(seedurl, 0)
}

func crawl (currenturl string, maxdepth int) {
	c := colly.NewCollector(
		colly.AllowedDomains("www.scrapingcourse.com"),
				colly.MaxDepth(maxdepth),
	)

	c.OnHTML("title", func(e *colly.HTMLElement) {
		fmt.Println("Page Title:", e.Text)
	})

	c.OnHTML("li.product", func(e *colly.HTMLElement) {
		productName := e.ChildText("h2")
		productPrice := e.ChildText("span.product-price")
		imageURL := e.ChildAttr("img", "src")

		product := Product{
			Name: productName,
			Price: productPrice,
			ImageURL: imageURL,

		}

		products = append(products, product)
	})

	c.OnHTML("a.page-numbers", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))

		if link != "" && !visitedurls[link] {

			visitedurls[link] = true
			fmt.Println("Found link:", link)

			e.Request.Visit(link)
		}
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Data extraction complete", r.Request.URL)
		exportToCSV("products.csv")
	})


	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawling", r.URL)
	})

	c.OnError(func(e *colly.Response, err error) {
		fmt.Println("Request URL:", e.Request.URL, "failed with response:", e, "\nError:", err)
	})

	err := c.Visit(currenturl)
	if err != nil {
		fmt.Println("Error visiting page:", err)
	}

}

func exportToCSV(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Name", "Price", "Image URL"})

	for _, product := range products {
		writer.Write([]string{product.Name, product.Price, product.ImageURL})
	}
	fmt.Println("Product details exported to", filename)
}
