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
			if total == 5 {
				break
			}
		}
	}

	if total == 0 {
		embed := failEmbed("Error", fmt.Sprintf("No results found for %q", query))
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		})
	}

	var fields []discord.EmbedField

	for _, match := range append(fromTitle, fromDesc...) {
		fields = append(fields, discord.EmbedField{
			Name:  fmt.Sprintf("%s, %s", match.Title, match.Date),
			Value: fmt.Sprintf("*%s*\n%s\n%s", match.Authors, match.Summary, match.URL),
		})
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
