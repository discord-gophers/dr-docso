package main

import (
	"fmt"
	"log"

	"github.com/DiscordGophers/dr-docso/spec"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

func (b *botState) handleSpec(e *gateway.InteractionCreateEvent, d *discord.CommandInteractionData) {
	// only arg and required, always present
	query := d.Options[0].String()

	log.Printf("%s used spec(%q)", e.User.Tag(), query)

	if len(query) < 3 || len(query) > 60 {
		embed := failEmbed("Error", "Your query must be between 3 and 60 characters.")
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		})
		return
	}

	switch query {
	case "toc", "contents", "list":
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: spec.TOC,
		})
		return
	}

	nodes := spec.Cache.Search(query)

	if len(nodes) > 8 {
		nodes = nodes[:8]
	}
	if len(nodes) != 1 {
		results := "*No Results found*"

		if len(nodes) != 0 {
			results = "Matches:\n"
			for _, node := range nodes {
				results += node.Match()
			}
		}

		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags: api.EphemeralResponse,
				Embeds: &[]discord.Embed{failEmbed(
					"Error",
					fmt.Sprintf("An exact match was not found for %q.\n\nTry `/spec query:toc`.\n%s", query, results),
				)},
				Components: spec.NodesSelect(nodes),
			},
		})
		return
	}

	// TODO: Show components if more than one result.

	node := nodes[0]
	md, _ := node.Render(1000)

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

func (b *botState) handleSpecComponent(e *gateway.InteractionCreateEvent, data *discord.ComponentInteractionData, cmd string) {
	switch cmd {
	case "toc":
		opt := data.Values[0]

		if opt == "back" {
			b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
				Type: api.UpdateMessage,
				Data: spec.TOC,
			})
			return
		}

		node := spec.Cache.Headings[opt]
		md, _ := node.Render(1800)

		options, ok := spec.Subcomponents[opt]
		if !ok {
			options = []discord.SelectComponentOption{spec.GoBack}
		}

		err := b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.UpdateMessage,
			Data: &api.InteractionResponseData{
				Flags: api.EphemeralResponse,
				Embeds: &[]discord.Embed{
					{
						Title:       "Spec - " + opt,
						Description: md,
						Color:       accentColor,
					},
				},
				Components: &[]discord.Component{
					&discord.ActionRowComponent{
						Components: []discord.Component{
							&discord.SelectComponent{
								CustomID:    "spec.toc",
								Placeholder: "View Subheadings",
								Options:     options,
							},
						},
					},
				},
			},
		})
		if err != nil {
			fmt.Println(string(err.(*httputil.HTTPError).Body))
		}
	}
}
