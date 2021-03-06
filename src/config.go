package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config can be used to read the configuration of the software
type Config struct {
	HexEncodedDiscordPublicKey string
	ThisInstanceURL            string
	ClientID                   string
	ClientSecret               string
	Games                      map[string]string
	Events                     map[string]Event
}

type Event struct {
	FirstTime time.Time
	Repeat    string
}

func ReadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("could not open config file: %w", err)
	}
	config := Config{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, fmt.Errorf("could not parse config file: %w", err)
	}
	return config, nil
}
