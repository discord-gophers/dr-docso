package main

import (
	"fmt"
	"strings"

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
		Color: accentColor,
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
	}, dMore || cMore
}

func helpEmbed() discord.Embed {
	return discord.Embed{
		Title: "Docs help",
		Description: `Dr-Docso is a bot to query Go documentation.
The parsing is done using [hhhapz/doc](https://github.com/hhhapz/doc).

Here are some example queries:` + "```md" + `
# Docs help
/docs help

# List aliases
/docs alias

# Search a module
/docs query:fmt

# Search a type
/docs query:github.com/hhhapz/doc.package

# Search a type method
/docs query:github.com/hhhapz/doc searcher search

# Many standard library types have aliases
/docs query:http (-> net/http)
` + "```",
		Footer: &discord.EmbedFooter{Text: "Source Code: https://github.com/DiscordGophers/dr-docso"},
		Color:  accentColor,
	}
}

func aliasList(aliases map[string]string) discord.Embed {
	keys := make([]string, 0, len(aliases))
	for k, v := range aliases {
		keys = append(keys, fmt.Sprintf("%s -> %s", k, v))
	}

	desc := "*No aliases defined*"
	if len(keys) > 0 {
		desc = fmt.Sprintf("```fix\n%s```", strings.Join(keys, "\n"))
	}

	return discord.Embed{
		Title:       "Current aliases",
		Description: desc,
		Color:       accentColor,
	}
}

func ignoreList(blacklist map[discord.Snowflake]struct{}) discord.Embed {
	keys := make([]string, 0, len(blacklist))
	for k := range blacklist {
		keys = append(keys, fmt.Sprintf("- <@!%s>", k))
	}

	desc := "*No ignores set*"
	if len(keys) > 0 {
		desc = strings.Join(keys, "\n")
	}

	return discord.Embed{
		Title:       "Ignored Users",
		Description: desc,
		Color:       accentColor,
	}
}

func failEmbed(title, description string) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       0xEE0000,
	}
}
