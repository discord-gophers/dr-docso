package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/hhhapz/doc"
)

func (b *botState) handleConfig(e *gateway.InteractionCreateEvent, d *discord.CommandInteraction) {
	grp := d.Options[0]
	cmd := grp.Options[0]
	log.Printf("%s used config %s %s", e.User.Tag(), grp.Name, cmd.Name)

	var embed discord.Embed
block:
	switch grp.Name {
	case "user":
		switch cmd.Name {
		case "ignore":
			user, _ := cmd.Options[0].SnowflakeValue()

			if ok := b.canIgnore(e.GuildID, user); !ok {
				embed = failEmbed("Error", fmt.Sprintf("You cannot ignore <@!%s>.", user))
				break block
			}

			if _, ok := b.cfg.Blacklist[user]; ok {
				embed = failEmbed("Error", fmt.Sprintf("<@!%s> is already being ignored.", user))
				break block
			}

			b.cfg.Blacklist[user] = struct{}{}
			embed = discord.Embed{
				Title:       "Success",
				Description: fmt.Sprintf("<@!%s> is now going to be ignored from all commands on Dr-Docso.", user),
				Color:       accentColor,
			}

		case "unignore":
			user, _ := cmd.Options[0].SnowflakeValue()

			if _, ok := b.cfg.Blacklist[user]; !ok {
				embed = failEmbed("Error", fmt.Sprintf("<@!%s> is not being ignored.", user))
				break block
			}

			delete(b.cfg.Blacklist, user)
			embed = discord.Embed{
				Title:       "Success",
				Description: fmt.Sprintf("<@!%s> is now unignored.", user),
				Color:       accentColor,
			}

		case "ignorelist":
			embed = ignoreList(b.cfg.Blacklist)
		}

	case "cache":
		switch cmd.Name {
		case "remove":
			lower := strings.ToLower(cmd.Options[0].String())

			var items []string
			b.searcher.WithCache(func(cache map[string]*doc.CachedPackage) {
				for item := range cache {
					if strings.Contains(strings.ToLower(item), lower) {
						delete(cache, item)
						items = append(items, "- "+item)
					}
				}
			})

			list := strings.Join(items, "\n")
			if len(list) > 4000 {
				list = list[:3800] + "..."
			}
			if len(list) == 0 {
				list = "(empty)"
			}

			embed = discord.Embed{
				Title: "Removed packages",
				Description: fmt.Sprintf("Removed %d Item(s):```fix\n%s```",
					len(items), list),
				Color: accentColor,
			}

		case "prune":
			var items []string
			b.searcher.WithCache(func(cache map[string]*doc.CachedPackage) {
				for k, cp := range cache {
					if time.Since(cp.Created) > time.Hour*24 { // removed stuff not used in over 24 hours
						delete(cache, k)
						items = append(items, "- "+k)
					}
				}
			})

			list := strings.Join(items, "\n")
			if len(list) > 4000 {
				list = list[:3800] + "..."
			}
			if len(list) == 0 {
				list = "(empty)"
			}

			embed = discord.Embed{
				Title: "Pruned packages",
				Description: fmt.Sprintf("Pruned %d Item(s):```fix\n%s```",
					len(items), list),
				Color: accentColor,
			}
		}

	case "alias":
		switch cmd.Name {
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

	if err := b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags:  api.EphemeralResponse,
			Embeds: &[]discord.Embed{embed},
		},
	}); err != nil {
		log.Printf("could not send interaction callback, %v", err)
	}
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
