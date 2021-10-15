package main

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/dustin/go-humanize"
	"github.com/hhhapz/doc"
)

var started = time.Now().Unix()

func (b *botState) handleInfo(e *gateway.InteractionCreateEvent, _ *discord.CommandInteractionData) {
	log.Printf("%s used info", e.User.Tag())

	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	var items int
	b.searcher.WithCache(func(cache map[string]*doc.CachedPackage) {
		items = len(cache)
	})

	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "Go: %s\n", runtime.Version())
	fmt.Fprintf(buf, "Uptime: <t:%[1]d:R> (<t:%[1]d:F>)\n", started)
	fmt.Fprintf(buf, "Memory: %s / %s (alloc / sys)\n", humanize.Bytes(stats.Alloc), humanize.Bytes(stats.Sys))
	fmt.Fprintf(buf, "Source: %s\n", "[link](https://github.com/DiscordGophers/dr-docso)")
	fmt.Fprintf(buf, "Concurrent Tasks: %s\n", humanize.Comma(int64(runtime.NumGoroutine())))
	fmt.Fprintf(buf, "Cached Entries: %s\n\n", humanize.Comma(int64(items)))
	fmt.Fprintf(buf, "Maintained by: %s\n", "[hhhapz#8936](https://github.com/hhhapz)")
	fmt.Fprintf(buf, "Hosted on %s by %s!\n", "[TransIP](https://www.transip.nl/)", "[Sgt_Tailor#0124](https://github.com/svenwiltink)")

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags: api.EphemeralResponse,
			Embeds: &[]discord.Embed{{
				Title:       "Dr-Docso",
				Description: buf.String(),
				Color:       accentColor,
			}},
			Components: &[]discord.Component{
				&discord.ActionRowComponent{
					Components: []discord.Component{
						&discord.ButtonComponent{
							Label:    "Command Info",
							CustomID: "info.help",
							Style:    discord.SecondaryButton,
						},
					},
				},
			},
		},
	})
}

func (b *botState) handleInfoComponent(e *gateway.InteractionCreateEvent, data *discord.ComponentInteractionData, cmd string) {
	switch cmd {
	case "help":
		b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags: api.EphemeralResponse,
				Embeds: &[]discord.Embed{
					{
						Title: fmt.Sprintf("Command Help"),
						Fields: []discord.EmbedField{
							{
								Name:  "/docs",
								Value: "Query Go package documentation.\nSee options in autocomplete.",
							},
							{
								Name:  "d.docs <module> [item]",
								Value: "Query Go package documentation.\n*Text commeand version.*",
							},
							{
								Name:  "/blog <slug|query>",
								Value: "Query [Go Blog](https://go.dev/blog) articles.",
							},
							{
								Name:  "/info",
								Value: "Bot Information.",
							},
							{
								Name:  "/config",
								Value: "Configure dr-docso.\n*(Herders only)*",
							},
						},
						Color: accentColor,
					},
				},
			},
		})
	}
}
