package main

import (
	"fmt"
	"log"

	"github.com/DiscordGophers/dr-docso/spec"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

var specCache spec.Spec

func init() {
	var err error
	specCache, err = spec.QuerySpec()
	if err != nil {
		panic(err)
	}
}

func (b *botState) handleSpec(e *gateway.InteractionCreateEvent, d *discord.CommandInteractionData) {
	// only arg and required, always present
	query := d.Options[0].String()

	log.Printf("%s used spec(%q)", e.User.Tag(), query)

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

	nodes := specCache.Search(query)

	if len(nodes) != 1 {
		embed := failEmbed("Error", fmt.Sprintf("An exact match was not found for %q", query))
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		})
	}

	// TODO: Show components if more than one result.

	node := nodes[0]
	md, _ := node.Markdown(1000)

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Embeds: &[]discord.Embed{
				{
					Title:       fmt.Sprintf("Spec: %q", query),
					Description: md,
					Color:       accentColor,
				},
			},
		},
	})
}
