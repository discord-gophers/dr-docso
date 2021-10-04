package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/hhhapz/doc"
	"github.com/hhhapz/doc/godocs"
)

func run() error {
	cfg := config()
	if cfg.Token == "" {
		return fmt.Errorf("no token provided")
	}

	s, err := state.New("Bot " + cfg.Token)
	if err != nil {
		return fmt.Errorf("could not open session: %w", err)
	}

	searcher := doc.New(http.DefaultClient, godocs.Parser, doc.UserAgent("https://github.com/DiscordGophers/dr-docso (Discord Server for Go)"))
	b := botState{
		cfg:      cfg,
		searcher: doc.WithCache(searcher),
		state:    s,
	}

	s.AddHandler(b.OnCommand)
	s.AddHandler(b.OnMessage)
	s.AddHandler(b.OnMessageEdit)
	s.AddIntents(gateway.IntentGuildMessages | gateway.IntentGuildMessageReactions)

	if err := s.Open(context.Background()); err != nil {
		return fmt.Errorf("failed to open: %w", err)
	}
	defer s.Close()

	log.Println("Gateway connection established.")

	me, err := s.Me()
	if err != nil {
		return fmt.Errorf("could not get me: %w", err)
	}
	b.appID = discord.AppID(me.ID)

	if err := loadCommands(s, me.ID, cfg); err != nil {
		return fmt.Errorf("could not init commands: %w", err)
	}

	log.Println("Logged in as ", me.Tag())

	go b.gcInteractionData()
	go b.updateArticles()
	select {}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}
