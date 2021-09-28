package blog

import (
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
)

type Article struct {
	Title      string
	titleLower string

	URL     string
	Date    string
	Authors string

	Summary      string
	summaryLower string

	Slug string
}

type MatchType uint8

const (
	NoMatch MatchType = iota
	MatchTitle
	MatchDesc
	MatchExact
)

func MatchAll(articles []Article, keyword string) (title []Article, desc []Article, total int) {
	for _, a := range articles {
		switch a.Match(keyword) {
		case NoMatch:
			continue
		case MatchExact:
			return []Article{a}, nil, 1
		case MatchTitle, MatchDesc:
			title = append(title, a)
		default:
			continue
		}
		total++
	}
	return
}

func (a Article) Match(keyword string) MatchType {
	if a.Slug == keyword {
		return MatchExact
	}

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
		return NoMatch
	}
	return match
}

func (a Article) Display() discord.Embed {
	return discord.Embed{
		Title:       a.Title,
		URL:         a.URL,
		Description: a.Summary,
		Footer: &discord.EmbedFooter{
			Text: a.Authors + "\n" + a.Date,
		},
		Color: 0x00ADD8,
	}
}
