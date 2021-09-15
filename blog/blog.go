package blog

import "strings"

type Article struct {
	Title      string
	titleLower string

	URL     string
	Date    string
	Authors string

	Summary      string
	summaryLower string
}

func (a Article) Match(keyword string) bool {
	f := strings.Fields(strings.ToLower(keyword))
	for _, s := range f {
		if strings.Contains(a.titleLower, s) {
			continue
		}
		if strings.Contains(a.summaryLower, s) {
			continue
		}
		return false
	}
	return true
}
