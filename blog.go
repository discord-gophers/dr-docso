package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DiscordGophers/dr-docso/blog"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
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
	BlogNoResults = "No results found for %q\n\n To match blog titles and descriptions, try enabling the matchDesc parameter."
)

func (b *botState) handleBlog(e *gateway.InteractionCreateEvent, d *discord.CommandInteractionData) {
	// only arg and required, always present
	query := d.Options[0].String()

	var matchDesc bool
	if len(d.Options) > 1 {
		matchDesc, _ = d.Options[0].Bool()
	}

	log.Printf("%s used blog(%q, %t)", e.User.Tag(), query, matchDesc)

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

	var fromTitle, fromDesc []blog.Article
	var total int

	for _, a := range b.articles {
		if ok, typ := a.Match(query); ok {
			switch {
			case typ == blog.MatchTitle:
				fromTitle = append(fromTitle, a)
			case typ == blog.MatchDesc && matchDesc:
				fromDesc = append(fromDesc, a)
			default:
				continue
			}

			total++
			if total == 20 {
				break
			}
		}
	}

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

	var fields []discord.EmbedField
	var components []discord.SelectComponentOption
	for _, match := range append(fromTitle, fromDesc...) {
		fields = append(fields, discord.EmbedField{
			Name:  fmt.Sprintf("%s, %s", match.Title, match.Date),
			Value: fmt.Sprintf("*%s*\n%s\n%s", match.Authors, match.Summary, match.URL),
		})
		components = append(components, discord.SelectComponentOption{
			Label:       match.Title,
			Value:       match.URL,
			Description: match.Date + " - " + match.Authors,
		})
	}

	if len(fields) > 2 {
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags: api.EphemeralResponse,
				Embeds: &[]discord.Embed{
					{
						Title: fmt.Sprintf("Blog: %d Results", total),
						Description: fmt.Sprintf(
							"Search Term: %q\nMatch on description: %t\nTo display publicly, select single post:",
							query, matchDesc,
						),
						Fields: fields,
						Color:  accentColor,
					},
				},
				Components: &[]discord.Component{
					&discord.ActionRowComponent{
						Components: []discord.Component{
							&discord.SelectComponent{
								CustomID:    "blog.display",
								Options:     components,
								Placeholder: "Display Blog Post",
							},
						},
					},
				},
			},
		})
		return
	}

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Embeds: &[]discord.Embed{
				{
					Title:       fmt.Sprintf("Blog: %d Results", total),
					Description: fmt.Sprintf("Search Term: %q\nMatch on description: %t", query, matchDesc),
					Fields:      fields,
					Color:       accentColor,
				},
			},
		},
	})
}

func (b *botState) handleBlogComponent(e *gateway.InteractionCreateEvent, data *discord.ComponentInteractionData, cmd string) {
	switch cmd {
	case "display":
		// should be url to post
		opt := data.Values[0]

		var article blog.Article
		for _, a := range b.articles {
			if a.URL == opt {
				article = a
			}
		}

		if article.URL == "" {
			return
		}

		err := b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Embeds: &[]discord.Embed{
					{
						Title:       article.Title,
						URL:         article.URL,
						Description: article.Summary,
						Footer: &discord.EmbedFooter{
							Text: article.Authors + "\n" + article.Date,
						},
						Color: accentColor,
					},
				},
			},
		})
		if err != nil {
			log.Printf("Could not respond to blog interaction: %v", err)
		}
	}
}
