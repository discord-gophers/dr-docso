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

type MatchType uint8

const (
	MatchTitle MatchType = iota
	MatchDesc
)

func (a Article) Match(keyword string) (bool, MatchType) {
	f := strings.Fields(strings.ToLower(keyword))

	match := MatchDesc

	for _, s := range f {
		if strings.Contains(a.titleLower, s) {
			match = MatchTitle
			continue
		}
		if strings.Contains(a.summaryLower, s) {
			continue
		}
		return false, 0
	}
	return true, match
}
