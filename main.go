package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/DiscordGophers/dr-docso/blog"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/hhhapz/doc"
	"github.com/hhhapz/doc/godocs"
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
		case "info":
			b.handleInfo(e, data)
		case "config":
			b.handleConfig(e, data)
		}

	case *discord.ComponentInteractionData:
		if d, ok := interactionMap[data.CustomID]; ok {
			b.onDocsComponent(e, d)
			return
		}
	}
}

var update bool

func main() {
	updateVar := flag.Bool("update", false, "update all commands, regardless of if they are present or not")
	flag.Parse()
	update = *updateVar

	cfg := config()
	if cfg.Token == "" {
		log.Fatal("no token provided")
	}

	s, err := state.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatalln(fmt.Errorf("could not open session, %w", err))
	}

	searcher := doc.New(http.DefaultClient, godocs.Parser)
	b := botState{
		cfg:      cfg,
		searcher: doc.WithCache(searcher),
		state:    s,
	}

	s.AddHandler(b.OnCommand)
	s.AddIntents(gateway.IntentGuildMessageReactions)

	if err := s.Open(context.Background()); err != nil {
		log.Fatalln("failed to open:", err)
	}
	defer s.Close()

	log.Println("Gateway connection established.")
	me, err := s.Me()
	if err != nil {
		log.Println("Could not get me:", err)
		return
	}
	b.appID = discord.AppID(me.ID)

	if err := loadCommands(s, me.ID, cfg); err != nil {
		log.Println("Could not load commands:", err)
		return
	}

	log.Println("Logged in as ", me.Tag())

	go b.gcInteractionData()
	go b.updateArticles()
	select {}
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
			{
				Name:        "match-desc",
				Description: "Match on blog description as well as title",
				Type:        discord.BooleanOption,
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
