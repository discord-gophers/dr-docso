package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/hhhapz/discodoc/blog"
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

func (b *botState) handleBlog(e *gateway.InteractionCreateEvent, d *discord.CommandInteractionData) {
	// only arg and required, always present
	query := d.Options[0].String()

	log.Printf("%s used blog(%q)", e.User.Tag(), query)

	if len(query) < 3 || len(query) > 20 {
		embed, _ := failEmbed("Error", "Your query must be between 3 and 20 characters.")
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		})
		return
	}

	var matched []blog.Article
	for _, a := range b.articles {
		if a.Match(query) {
			matched = append(matched, a)
			if len(matched) == 5 {
				break
			}
		}
	}

	embed := discord.Embed{
		Title:       fmt.Sprintf("Blog: %d Results", len(matched)),
		Description: fmt.Sprintf("Search Term: %q", query),
	}
	for _, match := range matched {
		embed.Fields = append(embed.Fields, discord.EmbedField{
			Name:  fmt.Sprintf("%s, %s", match.Title, match.Date),
			Value: fmt.Sprintf("*%s*\n%s\n%s", match.Authors, match.Summary, match.URL),
		})
	}

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Embeds: &[]discord.Embed{embed},
		},
	})
}
