package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/hhhapz/doc"
	"github.com/hhhapz/doc/godocs"
	"github.com/pkg/errors"
)

type botState struct {
	cfg      configuration
	appID    discord.AppID
	searcher *doc.CachedSearcher
	state    *state.State
}

func (b *botState) OnCommand(e *gateway.InteractionCreateEvent) {
	if e.GuildID != 0 {
		e.User = &e.Member.User
	}

	switch data := e.Data.(type) {
	case *discord.CommandInteractionData:
		switch data.Name {
		case "docs":
			b.handleDocs(e, data)
		case "info":
			b.handleInfo(e, data)
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
		log.Fatalln(errors.Wrap(err, "could not open session"))
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

	log.Println("Logged in as ", me.Tag())

	if err := loadCommands(s, me.ID, cfg); err != nil {
		log.Println("Could not load commands:", err)
		return
	}

	go b.gcInteractionData()
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
		var err error
		if _, err = s.CreateCommand(appID, c); err != nil {
			return errors.Wrap(err, "could not register "+c.Name)
		}
		log.Println("Created command:", c.Name)
	}

	return nil
}

var commands = []api.CreateCommandData{
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
		Name:        "go",
		Description: "Helpful Go Macros",
		Options: []discord.CommandOption{
			{
				Name:        "keyword",
				Description: "Keyword",
				Type:        discord.StringOption,
				Required:    true,
			},
		},
	},
	{
		Name:                "modgo",
		Description:         "Modify Go Macros",
		NoDefaultPermission: true,
		Options: []discord.CommandOption{
			{
				Name:        "add",
				Description: "Add a macro",
				Type:        discord.SubcommandOption,
				Options: []discord.CommandOption{
					{
						Name:        "url",
						Description: "URL",
						Type:        discord.StringOption,
						Required:    true,
					},
					{
						Name:        "keywords",
						Description: "Keywords (Specify multiple with space)",
						Type:        discord.StringOption,
						Required:    true,
					},
					{
						Name:        "title",
						Description: "Title",
						Type:        discord.StringOption,
						Required:    true,
					},
				},
			},
			{
				Name:        "rm",
				Description: "Remove a macro",
				Type:        discord.SubcommandOption,
				Options: []discord.CommandOption{
					{
						Name:        "url",
						Description: "URL",
						Type:        discord.StringOption,
						Required:    true,
					},
				},
			},
		},
	},
	{
		Name:        "info",
		Description: "Generic Bot Info",
	},
}
