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
	Events                 []RsvpEvent
}

// RsvpEvent stores the webhook message ids for an event that is currently in the rsvp phase.
type RsvpEvent struct {
	Title          string
	StartsAt       time.Time
	WebhookID      string
	WebhookToken   string
	TitleMessageID string
	GameMessageIDs map[string]string
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

func (s *State) AddRsvpEvent(event RsvpEvent) {
	s.Events = append(s.Events, event)
	s.save()
}

func (s *State) RemoveRsvpEvent(title string, startsAt time.Time) {
	// find the index of the event to delete it
	index := -1
	for i, event := range s.Events {
		if event.Title == title && event.StartsAt == startsAt {
			index = i
		}
	}

	if index >= 0 {
		if index+1 == len(s.Events) {
			s.Events = s.Events[:index]
		} else {
			s.Events = append(s.Events[:index], s.Events[index+1:]...)
		}
		s.save()
	}
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
