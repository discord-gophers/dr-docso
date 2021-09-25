package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DiscordGophers/dr-docso/blog"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

func (b *botState) updateArticles() {
	articles, err := blog.Articles(http.DefaultClient)
	if err != nil {
		panic(err)
	}
	b.articles = articles

	articleTicker := time.NewTicker(time.Hour * 72)
	for {
		select {
		case <-articleTicker.C:
			articles, err := blog.Articles(http.DefaultClient)
			if err != nil {
				log.Printf("Error querying maps: %v", err)
				continue
			}

			b.articles = articles
		}
	}
}

const (
	BlogNoResults = "No results found for %q""
)

func (b *botState) handleBlog(e *gateway.InteractionCreateEvent, d *discord.CommandInteractionData) {
	// only arg and required, always present
	query := d.Options[0].String()

	log.Printf("%s used blog(%q)", e.User.Tag(), query)

	if len(query) < 3 || len(query) > 20 {
		embed := failEmbed("Error", "Your query must be between 3 and 20 characters.")
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		})
		return
	}

	fromTitle, fromDesc, total := blog.MatchAll(b.articles, query)
	if total == 0 {
		embed := failEmbed("Error", fmt.Sprintf(BlogNoResults, query))
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		})
	}
	articles := append(fromTitle, fromDesc...)

	fields, opts := articleFields(articles)

	if total <= 2 {
		embed := discord.Embed{
			Title:  fmt.Sprintf("Blog: %q", query),
			Fields: fields,
			Color:  accentColor,
		}
		if total == 1 {
			embed = articles[0].Display()
		}
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Embeds: &[]discord.Embed{embed},
			},
		})
		return
	}

	if len(opts) > 5 {
		opts = opts[:5]
	}
	comps := []discord.Component{
		&discord.ActionRowComponent{
			Components: []discord.Component{
				&discord.SelectComponent{
					CustomID:    "blog.display",
					Options:     opts,
					Placeholder: "Display Blog Post",
				},
			},
		},
	}

	if len(fields) > 5 {
		fields = fields[:5]
		comps = append(comps, &discord.ActionRowComponent{
			Components: []discord.Component{
				&discord.ButtonComponent{
					Label:    "Prev Page",
					CustomID: "blog.prev." + query,
					Style:    discord.SecondaryButton,
					Emoji:    &discord.ButtonEmoji{Name: "⬅️"},
				},
				&discord.ButtonComponent{
					Label:    "Next Page",
					CustomID: "blog.next." + query,
					Style:    discord.SecondaryButton,
					Emoji:    &discord.ButtonEmoji{Name: "➡️"},
				},
			},
		})
	}

	p := int(math.Ceil(float64(total) / float64(5)))
	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags: api.EphemeralResponse,
			Embeds: &[]discord.Embed{
				{
					Title: fmt.Sprintf("Blog: %d Results", total),
					Footer: &discord.EmbedFooter{
						Text: fmt.Sprintf("Page 1 of %d\nTo display publicly, select a single post", p),
					},
					Fields: append([]discord.EmbedField{
						{
							Name:  "Search Term",
							Value: query,
						},
					}, fields...),
					Color: accentColor,
				},
			},
			Components: &comps,
		},
	})
}

func (b *botState) handleBlogComponent(e *gateway.InteractionCreateEvent, data *discord.ComponentInteractionData, cmd string) {
	switch cmd {
	case "display":
		b.BlogDisplay(e, data.Values[0])
		return
	}

	split := strings.SplitN(cmd, ".", 2)
	query := split[1]

	embed := e.Message.Embeds[0]
	matches := pageRe.FindStringSubmatch(embed.Footer.Text)
	if len(matches) != 3 {
		return
	}

	cur, _ := strconv.Atoi(matches[1])

	switch split[0] {
	case "prev":
		cur--
	case "next":
		cur++
	}

	fromTitle, fromDesc, total := blog.MatchAll(b.articles, query)
	p := int(math.Ceil(float64(total) / float64(5)))
	if cur < 1 || cur > p {
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.UpdateMessage, Data: &api.InteractionResponseData{},
		})
		return
	}
	fields, opts := articleFields(append(fromTitle, fromDesc...))
	if cur != p {
		fields = fields[(cur-1)*5 : cur*5]
		opts = opts[(cur-1)*5 : cur*5]
	} else {
		fields = fields[(cur-1)*5:]
		opts = opts[(cur-1)*5:]
	}

	comps := []discord.Component{
		&discord.ActionRowComponent{
			Components: []discord.Component{
				&discord.SelectComponent{
					CustomID:    "blog.display",
					Options:     opts,
					Placeholder: "Display Blog Post",
				},
			},
		},
		&discord.ActionRowComponent{
			Components: []discord.Component{
				&discord.ButtonComponent{
					Label:    "Prev Page",
					CustomID: "blog.prev." + query,
					Style:    discord.SecondaryButton,
					Emoji:    &discord.ButtonEmoji{Name: "⬅️"},
				},
				&discord.ButtonComponent{
					Label:    "Next Page",
					CustomID: "blog.next." + query,
					Style:    discord.SecondaryButton,
					Emoji:    &discord.ButtonEmoji{Name: "➡️"},
				},
			},
		},
	}

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.UpdateMessage,
		Data: &api.InteractionResponseData{
			Flags: api.EphemeralResponse,
			Embeds: &[]discord.Embed{
				{
					Title: fmt.Sprintf("Blog: %d Results", total),
					Footer: &discord.EmbedFooter{
						Text: fmt.Sprintf("Page %d of %d\nTo display publicly, select a single post", cur, p),
					},
					Fields: append([]discord.EmbedField{
						{
							Name:  "Search Term",
							Value: query,
						},
					}, fields...),
					Color: accentColor,
				},
			},
			Components: &comps,
		},
	})
}

var pageRe = regexp.MustCompile(`Page (\d+) of (\d+)`)

func (b *botState) BlogDisplay(e *gateway.InteractionCreateEvent, url string) {
	var article blog.Article
	for _, a := range b.articles {
		if a.URL == url {
			article = a
		}
	}

	if article.URL == "" {
		return
	}

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.UpdateMessage,
		Data: &api.InteractionResponseData{
			Components: &[]discord.Component{},
		},
	})
	b.state.CreateInteractionFollowup(e.AppID, e.Token, api.InteractionResponseData{
		Content: option.NewNullableString(e.User.Tag() + ":"),
		Embeds:  &[]discord.Embed{article.Display()},
	})
}

func articleFields(articles []blog.Article) (fields []discord.EmbedField, opts []discord.SelectComponentOption) {
	for _, a := range articles {
		fields = append(fields, discord.EmbedField{
			Name:  fmt.Sprintf("%s, %s", a.Title, a.Date),
			Value: fmt.Sprintf("*%s*\n%s\n%s", a.Authors, a.Summary, a.URL),
		})

		if len(a.Title) > 100 {
			fmt.Println(a.Title)
		}
		if len(a.URL) > 100 {
			fmt.Println(a.URL)
		}
		if len(a.Authors) > 100 {
			fmt.Println(a.Authors)
		}
		opts = append(opts, discord.SelectComponentOption{
			Label:       a.Title,
			Value:       a.URL,
			Description: a.Authors,
		})
	}
	return
}
