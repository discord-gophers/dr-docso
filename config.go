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
}

type commandPermissions struct {
	Docs map[discord.RoleID]bool `json:"docs"`
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
	return config
}
