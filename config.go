package main

import (
	"encoding/json"
	"log"
	"os"
)

type Configuration struct {
	Prefix string `json:"prefix"`
	Token  string `json:"token"`
}

var config Configuration

func init() {
	fileBytes, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(fileBytes, &config)
	if err != nil {
		log.Fatal(err)
	}
}
