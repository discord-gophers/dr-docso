package main

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

func (b *botState) handleConfig(e *gateway.InteractionCreateEvent, d *discord.CommandInteractionData) {
	// only arg and required, always present

	var embed discord.Embed

block:
	switch grp := d.Options[0]; grp.Name {
	case "user":
		switch cmd := grp.Options[0]; cmd.Name {
		case "ignore":
			user, _ := cmd.Options[0].Snowflake()

			if ok := b.canIgnore(e.GuildID, user); !ok {
				embed = failEmbed("Error", fmt.Sprintf("You cannot ignore <@!%s>.", user))
				break block
			}

			if _, ok := b.cfg.Blacklist[user]; ok {
				embed = failEmbed("Error", fmt.Sprintf("<@!%s> is already benig ignored.", user))
				break block
			}

			b.cfg.Blacklist[user] = struct{}{}
			embed = discord.Embed{
				Title:       "Success",
				Description: fmt.Sprintf("<@!%s> is now going to be ignored from all commands on discodoc.", user),
				Color:       accentColor,
			}

		case "unignore":
			user, _ := cmd.Options[0].Snowflake()

			if _, ok := b.cfg.Blacklist[user]; !ok {
				embed = failEmbed("Error", fmt.Sprintf("<@!%s> is not benig ignored.", user))
				break block
			}

			delete(b.cfg.Blacklist, user)
			embed = discord.Embed{
				Title:       "Success",
				Description: fmt.Sprintf("<@!%s> is now unignored.", user),
				Color:       accentColor,
			}
		}

	case "alias":
		switch cmd := grp.Options[0]; cmd.Name {
		case "add":
			alias := cmd.Options[0].String()
			keyword := cmd.Options[1].String()

			if strings.ContainsAny(alias, " .@/") {
				embed = failEmbed("Error", "Your alias contains illegal characters.")
				break block
			}

			b.cfg.Aliases[alias] = keyword
			embed = discord.Embed{
				Title:       "Success",
				Description: fmt.Sprintf("Searching module **%s** will now point to `%s`.", alias, keyword),
				Color:       accentColor,
			}
		case "remove":
			alias := cmd.Options[0].String()
			delete(b.cfg.Aliases, alias)
			embed = discord.Embed{
				Title:       "Success",
				Description: fmt.Sprintf("The `%s` alias has now been removed.", alias),
				Color:       accentColor,
			}
		case "list":
			embed = aliasList(b.cfg.Aliases)
		}
	}
	if !strings.HasPrefix(embed.Title, "Error") {
		err := saveConfig(b.cfg)
		if err != nil {
			embed = failEmbed("Error", fmt.Sprintf("Could not save config: `%v`", err))
		}
	}

	b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags:  api.EphemeralResponse,
			Embeds: &[]discord.Embed{embed},
		},
	})
}

func (b *botState) canIgnore(guild discord.GuildID, user discord.Snowflake) bool {
	m, err := b.state.Member(guild, discord.UserID(user))
	if err != nil {
		return false
	}
	for _, role := range m.RoleIDs {
		if _, ok := b.cfg.Permissions.Config[guild][discord.Snowflake(role)]; ok {
			return false
		}
	}
	return true
}
