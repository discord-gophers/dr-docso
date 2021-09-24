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
}

type MatchType uint8

const (
	MatchTitle MatchType = iota
	MatchDesc
)

func MatchAll(articles []Article, keyword string) (title []Article, desc []Article, total int) {
	for _, a := range articles {
		if ok, typ := a.Match(keyword); ok {
			total++
			switch {
			case typ == MatchTitle:
				title = append(title, a)
			case typ == MatchDesc:
				desc = append(desc, a)
			default:
				continue
			}
		}
	}
	return
}

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
