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
)

var started = time.Now().Unix()

func (b *botState) handleInfo(e *gateway.InteractionCreateEvent, _ *discord.CommandInteractionData) {
	log.Printf("%s used info", e.User.Tag())

	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "Go: %s\n", runtime.Version())
	fmt.Fprintf(buf, "Uptime: <t:%d:R>\n", started)
	fmt.Fprintf(buf, "Memory: %s / %s (alloc / sys)\n", humanize.Bytes(stats.Alloc), humanize.Bytes(stats.Sys))
	fmt.Fprintf(buf, "Source: %s\n", "[link](https://github.com/DiscordGophers/dr-docso)")
	fmt.Fprintf(buf, "Concurrent Tasks: %s\n\n", humanize.Comma(int64(runtime.NumGoroutine())))
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
		},
	})
}
