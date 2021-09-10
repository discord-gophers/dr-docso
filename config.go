package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/pkg/errors"
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
		log.Fatal(errors.Wrap(err, "could not open config"))
	}
	err = json.Unmarshal(fileBytes, &config)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not parse config"))
	}
	return config
}
