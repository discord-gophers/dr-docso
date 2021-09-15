package main

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/hhhapz/doc"
)

const (
	docLimit = 2800
	defLimit = 1000

	accentColor = 0x007D9C
)

func pkgEmbed(pkg doc.Package, full bool) (discord.Embed, bool) {
	c, more := comment(pkg.Overview, 32, full)
	return discord.Embed{
		Title: "Package " + pkg.URL,
		URL:   "https://pkg.go.dev/" + pkg.URL,
		Description: fmt.Sprintf("**Types:** %d\n**Functions:** %d\n\n%s",
			len(pkg.Types), len(pkg.Functions), c),
		Color:  accentColor,
		Footer: &discord.EmbedFooter{Text: "https://pkg.go.dev/" + pkg.URL},
	}, more
}

func typEmbed(pkg doc.Package, typ doc.Type, full bool) (discord.Embed, bool) {
	def, dMore := typdef(typ.Signature, full)
	c, cMore := comment(typ.Comment, len(def), full)
	return discord.Embed{
		Title:       fmt.Sprintf("%s: %s", pkg.URL, typ.Name),
		URL:         fmt.Sprintf("https://pkg.go.dev/%s#%s", pkg.URL, typ.Name),
		Description: fmt.Sprintf("```go\n%s\n```\n%s", def, c),
		Color:       accentColor,
		Footer:      &discord.EmbedFooter{Text: "https://pkg.go.dev/" + pkg.URL},
	}, dMore || cMore
}

func fnEmbed(pkg doc.Package, fn doc.Function, full bool) (discord.Embed, bool) {
	def, dMore := typdef(fn.Signature, full)
	c, cMore := comment(fn.Comment, len(def), full)
	return discord.Embed{
		Title:       fmt.Sprintf("%s: %s", pkg.URL, fn.Name),
		URL:         fmt.Sprintf("https://pkg.go.dev/%s#%s", pkg.URL, fn.Name),
		Description: fmt.Sprintf("```go\n%s\n```\n%s", def, c),
		Color:       accentColor,
		Footer:      &discord.EmbedFooter{Text: "https://pkg.go.dev/" + pkg.URL},
	}, dMore || cMore
}

func methodEmbed(pkg doc.Package, method doc.Method, full bool) (discord.Embed, bool) {
	def, dMore := typdef(method.Signature, full)
	c, cMore := comment(method.Comment, len(def), full)
	return discord.Embed{
		Title:       fmt.Sprintf("%s: %s.%s", pkg.URL, method.For, method.Name),
		URL:         fmt.Sprintf("https://pkg.go.dev/%s#%s.%s", pkg.URL, method.For, method.Name),
		Description: fmt.Sprintf("```go\n%s\n```\n%s", def, c),
		Color:       accentColor,
		Footer:      &discord.EmbedFooter{Text: "https://pkg.go.dev/" + pkg.URL},
	}, dMore || cMore
}

func failEmbed(title, description string) (discord.Embed, bool) {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       0xEE0000,
	}, false
}
