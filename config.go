package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/discord"
)

type configuration struct {
	Token       string             `json:"token"`
	Permissions commandPermissions `json:"permissions"`

	Aliases   map[string]string              `json:"aliases"`
	Blacklist map[discord.Snowflake]struct{} `json:"blacklist"`
}

type commandPermissions struct {
	Docs   map[discord.RoleID]struct{}                        `json:"docs"`
	Config map[discord.GuildID]map[discord.Snowflake]struct{} `json:"config"`
}

func config() configuration {
	var config configuration
	fileBytes, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(fmt.Errorf("could not open config, %w", err))
	}
	err = json.Unmarshal(fileBytes, &config)
	if err != nil {
		log.Fatal(fmt.Errorf("could not parse config, %w", err))
	}

	if config.Aliases == nil {
		config.Aliases = map[string]string{}
	}
	if config.Blacklist == nil {
		config.Blacklist = map[discord.Snowflake]struct{}{}
	}

	return config
}

func saveConfig(config configuration) error {
	f, err := os.OpenFile("config.json", os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "\t")
	return encoder.Encode(config)
}
