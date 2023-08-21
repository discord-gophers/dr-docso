package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/DiscordGophers/dr-docso/blog"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/hhhapz/doc"
)

type botState struct {
	cfg      configuration
	appID    discord.AppID
	searcher *doc.CachedSearcher
	state    *state.State

	articles []blog.Article
}

func (b *botState) OnCommand(e *gateway.InteractionCreateEvent) {
	if e.GuildID != 0 {
		e.User = &e.Member.User
	}

	// ignore blacklisted users
	if _, ok := b.cfg.Blacklist[discord.Snowflake(e.User.ID)]; ok {
		log.Printf("Ignoring message from %s", e.User.Tag())
		return
	}

	switch data := e.Data.(type) {
	case *discord.AutocompleteInteraction:
		switch data.Name {
		case "docs":
			b.handleDocsComplete(e, data)
		}
	case *discord.CommandInteraction:
		switch data.Name {
		case "blog":
			b.handleBlog(e, data)
		case "docs":
			b.handleDocs(e, data)
		case "spec":
			b.handleSpec(e, data)
		case "info":
			b.handleInfo(e, data)
		case "config":
			b.handleConfig(e, data)
		}

	case discord.ComponentInteraction:
		if d, ok := interactionMap[string(data.ID())]; ok {
			b.handleDocsComponent(e, d)
			return
		}

		split := strings.SplitN(string(data.ID()), ".", 2)
		switch split[0] {
		case "blog":
			b.handleBlogComponent(e, data, split[1])
		case "docs":
		case "spec":
			b.handleSpecComponent(e, data, split[1])
		case "info":
			b.handleInfoComponent(e, data, split[1])
		}
	}
}

var cmdre = regexp.MustCompile(`\$\[([\w\d/.]+)\]`)

func (b *botState) OnMessage(m *gateway.MessageCreateEvent) {
	if _, ok := b.cfg.Blacklist[discord.Snowflake(m.Author.ID)]; ok {
		return
	}

	var queries []string
	for _, v := range cmdre.FindAllStringSubmatch(m.Content, 3) {
		queries = append(queries, v[1])
	}

	b.handleDocsText(m, queries)
}

func (b *botState) OnMessageEdit(e *gateway.MessageUpdateEvent) {
	b.OnMessage((*gateway.MessageCreateEvent)(e))
	b.state.Unreact(e.ChannelID, e.ID, "😕")
}

func loadCommands(s *state.State, me discord.UserID, cfg configuration) error {
	appID := discord.AppID(me)

	for _, c := range commands {
		if _, err := s.CreateCommand(appID, c); err != nil {
			var httperr *httputil.HTTPError
			if errors.As(err, &httperr) {
				log.Println(string(httperr.Body))
			}
			return fmt.Errorf("could not register: %s, %w", c.Name, err)
		}
		log.Println("Created command:", c.Name)
	}

	return nil
}

var commands = []api.CreateCommandData{
	{
		Name:        "blog",
		Description: "Search go.dev Blog Posts",
		Options: []discord.CommandOption{
			&discord.StringOption{
				OptionName:  "query",
				Description: "Search query",
				Required:    true,
			},
		},
	},
	{
		Name:        "docs",
		Description: "Search Go Package Docs",
		Options: []discord.CommandOption{
			&discord.StringOption{
				OptionName:   "module",
				Description:  "Module name",
				Autocomplete: true,
				Required:     true,
			},
			&discord.StringOption{
				OptionName:   "item",
				Description:  "Search item in module",
				Autocomplete: true,
				Required:     true,
			},
		},
	},
	{
		Name:        "spec",
		Description: "Search Go Specification",
		Options: []discord.CommandOption{
			&discord.StringOption{
				OptionName:  "query",
				Description: "Search query",
				Required:    true,
			},
		},
	},
	{
		Name:        "info",
		Description: "Generic Bot Info",
	},
	{
		Name:                "config",
		Description:         "Configure Dr-Docso",
		NoDefaultPermission: true,
		Options: []discord.CommandOption{
			&discord.SubcommandGroupOption{
				OptionName:  "user",
				Description: "Manage user access to Dr-Docso",
				Subcommands: []*discord.SubcommandOption{
					{
						OptionName:  "ignore",
						Description: "Ignore commands and actions from a user",
						Options: []discord.CommandOptionValue{
							&discord.UserOption{
								OptionName:  "user",
								Description: "User to ignore",
								Required:    true,
							},
						},
					},
					{
						OptionName:  "unignore",
						Description: "Stop ignoring commands and actions from a user",
						Options: []discord.CommandOptionValue{
							&discord.UserOption{
								OptionName:  "user",
								Description: "User to unignore",
								Required:    true,
							},
						},
					},
					{
						OptionName:  "ignorelist",
						Description: "List all ignored users",
					},
				},
			},
			&discord.SubcommandGroupOption{
				OptionName:  "cache",
				Description: "Manage package cache",
				Subcommands: []*discord.SubcommandOption{
					{
						OptionName:  "remove",
						Description: "Remove cache for a specific module",
						Options: []discord.CommandOptionValue{
							&discord.StringOption{
								OptionName:  "module",
								Description: "Module name",
								Required:    true,
							},
						},
					},
					{
						OptionName:  "prune",
						Description: "Prune package cache not used in over 24 hours",
					},
				},
			},
			&discord.SubcommandGroupOption{
				OptionName:  "alias",
				Description: "Configure /docs aliases",
				Subcommands: []*discord.SubcommandOption{
					{
						OptionName:  "add",
						Description: "Add an alias",
						Options: []discord.CommandOptionValue{
							&discord.StringOption{
								OptionName:  "alias",
								Description: "Alias name",
								Required:    true,
							},
							&discord.StringOption{
								OptionName:  "url",
								Description: "Full module name",
								Required:    true,
							},
						},
					},
					{
						OptionName:  "remove",
						Description: "Remove an alias",
						Options: []discord.CommandOptionValue{
							&discord.StringOption{
								OptionName:  "alias",
								Description: "Alias name",
								Required:    true,
							},
						},
					},
					{
						OptionName:  "list",
						Description: "List all aliases",
					},
				},
			},
		},
	},
}
