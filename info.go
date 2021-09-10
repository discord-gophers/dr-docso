package main

import (
	"bytes"
	"fmt"
	"runtime"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/dustin/go-humanize"
)

var started = time.Now().Unix()

func (b *botState) handleInfo(e *gateway.InteractionCreateEvent, _ *discord.CommandInteractionData) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "Go: %s\n", runtime.Version())
	fmt.Fprintf(buf, "Uptime: <t:%d:R>\n", started)
	fmt.Fprintf(buf, "Memory: %s / %s (alloc / sys)\n", humanize.Bytes(stats.Alloc), humanize.Bytes(stats.Sys))
	fmt.Fprintf(buf, "Source: %s\n", "[link](https://gitea.teamortix.com/hamza/discodoc)")
	fmt.Fprintf(buf, "Concurrent Tasks: %s\n", humanize.Comma(int64(runtime.NumGoroutine())))

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags: api.EphemeralResponse,
			Embeds: &[]discord.Embed{{
				Title:       "DiscoDocs",
				Description: buf.String(),
				Color:       accentColor,
			}},
		},
	})
}
