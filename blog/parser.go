package blog

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	base     = "https://go.dev"
	articles = base + "/blog/all"
)

func Articles(client *http.Client) ([]Article, error) {
	res, err := client.Get(articles)
	if err != nil {
		return nil, fmt.Errorf("could not get articles: %w", err)
	}
	document, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not parse body: %w", err)
	}

	var articles []Article
	var article Article

	document.Find(".blogtitle, .blogsummary").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		class, _ := s.Attr("class")
		switch class {
		case "blogtitle":
			a := s.Find("a")
			uri := a.AttrOr("href", "")

			article = Article{
				Title:   a.Text(),
				URL:     base + uri,
				Date:    s.Find(".date").Text(),
				Authors: s.Find(".author").Text(),
				Slug:    path.Base(uri),
			}
			if article.Authors == "" {
				article.Authors = "No authors specified"
			}

			article.titleLower = strings.ToLower(article.Title)

		case "blogsummary":
			article.Summary = strings.TrimSpace(s.Text())
			article.summaryLower = strings.ToLower(article.Summary)
			articles = append(articles, article)
		}
		return true
	})

	return articles, nil
}
