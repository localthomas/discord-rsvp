package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

const stateFileDir = "./data/"
const stateFileName = "state.json"
const stateFilePath = stateFileDir + stateFileName

// State stores the application state. DO NOT USE FIELDS DIRECTLY!
type State struct {
	AuthorizationTokenType string
	AuthorizationToken     string
	ExpiresAt              time.Time
	RefreshToken           string
	WebhookID              string
	WebhookToken           string
}

func ResumeState() State {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return State{}
	}
	state := State{}
	err = json.Unmarshal(data, &state)
	if err != nil {
		return State{}
	}
	return state
}

func (s *State) SetToken(tokenType, token string, expiresAt time.Time, refreshToken string) {
	s.AuthorizationTokenType = tokenType
	s.AuthorizationToken = token
	s.ExpiresAt = expiresAt
	s.RefreshToken = refreshToken
	s.save()
}

func (s *State) SetWebhook(id, token string) {
	s.WebhookID = id
	s.WebhookToken = token
	s.save()
}

func (s *State) save() {
	err := os.MkdirAll(stateFileDir, os.ModeDir|0700)
	if err != nil {
		log.Fatalf("could not create state directory: %v", err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		log.Fatalf("could not marshal state: %v", err)
	}
	err = os.WriteFile(stateFilePath, data, 0700)
	if err != nil {
		log.Fatalf("could not save state: %v", err)
	}
}
