package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
)

type configuration struct {
	Prefix      string             `json:"prefix"`
	Token       string             `json:"token"`
	Permissions commandPermissions `json:"permissions"`

	Aliases map[string]string `json:"aliases"`

	Blacklist map[discord.Snowflake]struct{} `json:"blacklist"`
}

// snowflakeLookup transforms a json list to a map for faster lookups
type snowflakeLookup map[discord.Snowflake]struct{}

func (c *snowflakeLookup) UnmarshalJSON(data []byte) error {
	snowflakes := make([]discord.Snowflake, 0)

	err := json.Unmarshal(data, &snowflakes)
	if err != nil {
		return err
	}

	*c = map[discord.Snowflake]struct{}{}

	for _, s := range snowflakes {
		(*c)[s] = struct{}{}
	}

	return nil
}

func (c snowflakeLookup) MarshalJSON() ([]byte, error) {
	snowflakes := make([]discord.Snowflake, 0, len(c))
	for snowflake := range c {
		snowflakes = append(snowflakes, snowflake)
	}

	sort.Slice(snowflakes, func(i, j int) bool {
		return snowflakes[i] < snowflakes[j]
	})

	return json.Marshal(snowflakes)
}

type commandPermissions struct {
	Docs   snowflakeLookup                     `json:"docs"`
	Config map[discord.GuildID]snowflakeLookup `json:"config"`
}

func config() configuration {
	fileBytes, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(fmt.Errorf("could not open config, %w", err))
	}

	config, err := configFromBytes(fileBytes)
	if err != nil {
		log.Fatalf("could not parse config, %s", err)
	}

	if config.Aliases == nil {
		config.Aliases = map[string]string{}
	}
	if config.Blacklist == nil {
		config.Blacklist = map[discord.Snowflake]struct{}{}
	}

	return config
}

func configFromBytes(data []byte) (configuration, error) {
	var config configuration
	err := json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	if config.Aliases == nil {
		config.Aliases = map[string]string{}
	}
	if config.Blacklist == nil {
		config.Blacklist = map[discord.Snowflake]struct{}{}
	}

	return config, nil
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
