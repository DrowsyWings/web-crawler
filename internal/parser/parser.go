package parser

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Result struct {
	Title       string
	Description string
	Keywords    string
	Links       []string
}

func ParseHTML(url string, content string) (Result, error) {
	node, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return Result{}, err
	}

	var result Result
	result.Title = extractTitle(node)
	result.Links = extractLinks(node, url)
	result.Description, result.Keywords = extractMeta(node)

	return result, nil
}

func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
		return n.FirstChild.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		title := extractTitle(c)
		if title != "" {
			return title
		}

	}
	return ""
}

func extractMeta(n *html.Node) (string, string) {
	var description, keywords string
	if n.Type == html.ElementNode && n.Data == "meta" {
		var name, content string
		for _, attribute := range n.Attr {
			if attribute.Key == "name" {
				name = attribute.Val
			}
			if attribute.Key == "content" {
				content = attribute.Val
			}
		}
		name = strings.ToLower(name)
		if name == "description" {
			description = content
		} else if name == "keywords" {
			keywords = content
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		d, k := extractMeta(c)
		if d != "" {
			description = d
		}
		if k != "" {
			keywords = k
		}
	}
	return description, keywords
}

func extractLinks(n *html.Node, base string) []string {
	var links []string
	baseURL, _ := url.Parse(base)

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href := strings.TrimSpace(attr.Val)
					u, err := url.Parse(href)
					if err != nil {
						return
					}
					abs := baseURL.ResolveReference(u)
					if abs.Host == baseURL.Host {
						links = append(links, abs.String())
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return links
}
