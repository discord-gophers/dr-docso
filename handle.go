package main

import (
	"fmt"
	"log"
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
	case *discord.CommandInteractionData:
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

	case *discord.ComponentInteractionData:
		if d, ok := interactionMap[data.CustomID]; ok {
			b.handleDocsComponent(e, d)
			return
		}

		split := strings.SplitN(data.CustomID, ".", 2)
		switch split[0] {
		case "blog":
			b.handleBlogComponent(e, data, split[1])
		case "docs":
		case "spec":
			b.handleSpecComponent(e, data, split[1])
		}
	}
}

func (b *botState) OnMessage(m *gateway.MessageCreateEvent) {
	if b.cfg.Prefix == "" {
		return
	}

	c := m.Content
	if !strings.HasPrefix(c, b.cfg.Prefix) {
		return
	}

	if _, ok := b.cfg.Blacklist[discord.Snowflake(m.Author.ID)]; ok {
		log.Printf("Ignoring message from %s", m.Author.Tag())
		return
	}

	if m.GuildID != 0 {
		m.Author = m.Member.User
	}

	c = c[len(b.cfg.Prefix):]
	split := strings.SplitN(c, " ", 2)
	if len(split) != 2 {
		return
	}

	switch split[0] {
	case "docs":
		b.handleDocsText(m, split[1])
	case "help":
		// todo
	}
}

func (b *botState) OnMessageEdit(e *gateway.MessageUpdateEvent) {
	b.OnMessage((*gateway.MessageCreateEvent)(e))
}

func loadCommands(s *state.State, me discord.UserID, cfg configuration) error {
	appID := discord.AppID(me)
	registered, err := s.Commands(appID)
	if err != nil {
		return err
	}

	registeredMap := map[string]bool{}
	if !update {
		for _, c := range registered {
			registeredMap[c.Name] = true
			log.Println("Registered command:", c.Name)
		}
	}

	for _, c := range commands {
		if registeredMap[c.Name] {
			continue
		}
		var cmd *discord.Command
		var err error
		if cmd, err = s.CreateCommand(appID, c); err != nil {
			fmt.Println(string(err.(*httputil.HTTPError).Body))
			return fmt.Errorf("Could not register: %s, %w", c.Name, err)
		}

		switch c.Name {
		case "config":
			for guildID, opts := range cfg.Permissions.Config {
				var perms []discord.CommandPermissions
				for role := range opts {
					perms = append(perms, discord.CommandPermissions{
						ID:         role,
						Type:       discord.RoleCommandPermission,
						Permission: true,
					})
				}
				_, err := s.EditCommandPermissions(appID, guildID, cmd.ID, perms)
				if err != nil {
					fmt.Println(string(err.(*httputil.HTTPError).Message))
					return err
				}
			}
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
			{
				Name:        "query",
				Description: "Search query",
				Type:        discord.StringOption,
				Required:    true,
			},
		},
	},
	{
		Name:        "docs",
		Description: "Search Go Package Docs",
		Options: []discord.CommandOption{
			{
				Name:        "query",
				Description: "Search query (i.e strings.Split)",
				Type:        discord.StringOption,
				Required:    true,
			},
		},
	},
	{
		Name:        "spec",
		Description: "Search Go Specification",
		Options: []discord.CommandOption{
			{
				Name:        "query",
				Description: "Search query",
				Type:        discord.StringOption,
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
			{
				Type:        discord.SubcommandGroupOption,
				Name:        "user",
				Description: "Manage user access to Dr-Docso",
				Options: []discord.CommandOption{
					{
						Type:        discord.SubcommandOption,
						Name:        "ignore",
						Description: "Ignore commands and actions from a user",
						Options: []discord.CommandOption{
							{
								Type:        discord.UserOption,
								Name:        "user",
								Description: "User to ignore",
								Required:    true,
							},
						},
					},
					{
						Type:        discord.SubcommandOption,
						Name:        "unignore",
						Description: "Stop ignoring commands and actions from a user",
						Options: []discord.CommandOption{
							{
								Type:        discord.UserOption,
								Name:        "user",
								Description: "User to unignore",
								Required:    true,
							},
						},
					},
					{
						Type:        discord.SubcommandOption,
						Name:        "ignorelist",
						Description: "List all ignored users",
					},
				},
			},
			{
				Type:        discord.SubcommandGroupOption,
				Name:        "alias",
				Description: "Configure /docs aliases",
				Options: []discord.CommandOption{
					{
						Type:        discord.SubcommandOption,
						Name:        "add",
						Description: "Add an alias",
						Options: []discord.CommandOption{
							{
								Type:        discord.StringOption,
								Name:        "alias",
								Description: "Alias name",
								Required:    true,
							},
							{
								Type:        discord.StringOption,
								Name:        "url",
								Description: "Full module name",
								Required:    true,
							},
						},
					},
					{
						Type:        discord.SubcommandOption,
						Name:        "remove",
						Description: "Remove an alias",
						Options: []discord.CommandOption{
							{
								Type:        discord.StringOption,
								Name:        "alias",
								Description: "Alias name",
								Required:    true,
							},
						},
					},
					{
						Type:        discord.SubcommandOption,
						Name:        "list",
						Description: "List all aliases",
					},
				},
			},
		},
	},
}
