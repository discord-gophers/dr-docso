package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/pkg/errors"
)

type Configuration struct {
	Prefix string `json:"prefix"`
	Token  string `json:"token"`
}

func config() Configuration {
	var config Configuration
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
