package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/localthomas/discord-rsvp/discord"
)

func handleEventScheduling(session *discordgo.Session, state *State, config Config) {
	// eventsToCreate holds all possible events before checking if
	// they were already added to discord
	eventsToCreate := make(map[string][]time.Time)
	for eventTitle, eventData := range config.Events {
		eventsToCreate[eventTitle] = getPossibleTimes(eventData)
	}

	// check for already created events and remove them from the list eventsToCreate
	for eventTitle, eventTimes := range eventsToCreate {
		for index, eventTime := range eventTimes {
			wasAlreadyCreated := false
			for _, alreadyCreated := range state.Events {
				if alreadyCreated.Title == eventTitle && alreadyCreated.StartsAt == eventTime {
					wasAlreadyCreated = true
					break
				}
			}
			// if it was already created, remove the start time
			if wasAlreadyCreated {
				if index+1 == len(eventTimes) {
					eventsToCreate[eventTitle] = eventsToCreate[eventTitle][:index]
				} else {
					eventsToCreate[eventTitle] = append(
						eventsToCreate[eventTitle][:index],
						eventsToCreate[eventTitle][index+1:]...,
					)
				}
			}
		}
	}

	// add events
	for eventTitle, eventTimes := range eventsToCreate {
		for _, eventStartTime := range eventTimes {
			err := addEvent(session, state, config, eventTitle, eventStartTime)
			if err != nil {
				fmt.Printf("could not add event %v: %v\n", eventTitle, err)
			}
		}
	}

	// delete events that are in the past
	graceDuration := -2 * time.Hour
	for _, event := range state.Events {
		durationUntil := time.Until(event.StartsAt)
		if durationUntil < graceDuration {
			// event is in the past, delete it
			err := discord.DeleteWebhookMessage(session, event.WebhookID, event.WebhookToken, event.TitleMessageID)
			if err != nil {
				fmt.Printf("could not delete title message of event %v: %v\n", event.Title, err)
			}
			for gameTitle, messageID := range event.GameMessageIDs {
				err := discord.DeleteWebhookMessage(session, event.WebhookID, event.WebhookToken, messageID)
				if err != nil {
					fmt.Printf("could not delete game message of event %v with game %v: %v\n", event.Title, gameTitle, err)
				}
			}
			// propegate the change to the state
			state.RemoveRsvpEvent(event.Title, event.StartsAt)
		}
	}
}

func addEvent(
	session *discordgo.Session,
	state *State,
	config Config,
	eventTitle string,
	startTime time.Time,
) error {
	// one "title message" that contains info about the event itself
	webhookID := state.WebhookID
	webhookToken := state.WebhookToken
	message := createTitleMessage(eventTitle, startTime)
	messageReturn, err := discord.SendWebhookWithComponents(
		session,
		webhookID,
		webhookToken,
		true,
		message,
	)
	if err != nil {
		return fmt.Errorf("could not send title webhook message: %w", err)
	}
	titleMessageID := messageReturn.ID

	// create one message per game
	// if any error happens, the already created messages are deleted
	createdMessages := make(map[string]string)
	for gameTitle, gameDescription := range config.Games {
		message := createGameMessage(gameTitle, gameDescription)
		messageReturn, err := discord.SendWebhookWithComponents(
			session,
			webhookID,
			webhookToken,
			true,
			message,
		)
		if err != nil {
			// delete already created messages
			for _, messageID := range createdMessages {
				err = discord.DeleteWebhookMessage(session, webhookID, webhookToken, messageID)
				if err != nil {
					fmt.Printf("could not delete message prior to returning error: %v\n", err)
				}
			}
			return fmt.Errorf("could not send webhook message: %w", err)
		}
		createdMessages[gameTitle] = messageReturn.ID
	}

	state.AddRsvpEvent(RsvpEvent{
		TitleMessageID: titleMessageID,
		Title:          eventTitle,
		StartsAt:       startTime,
		WebhookID:      webhookID,
		WebhookToken:   webhookToken,
		GameMessageIDs: createdMessages,
	})
	return nil
}

func getPossibleTimes(eventData Event) []time.Time {
	// check for events in the near future (look ahead duration)
	lookAheadDuration := 5 * 24 * time.Hour
	times := make([]time.Time, 0)
	// check if there is the possibility that one of the repeating events
	// is in the look ahead duration
	possibleTimeToCheck := eventData.FirstTime
	for {
		durationUntil := time.Until(possibleTimeToCheck)
		if durationUntil > 0 &&
			// time.Until() can produce negative results,
			// so durationUntilWithoutLookAhead could end up being greater than the look ahead window
			durationUntil < lookAheadDuration {
			// timeToCheck is in the look ahead window, add to adding list
			times = append(times, possibleTimeToCheck)
		} else if durationUntil > lookAheadDuration {
			// the possibleTimeToCheck is so far into the future, that it even left the window
			break
		}

		switch eventData.Repeat {
		case "daily":
			possibleTimeToCheck = possibleTimeToCheck.AddDate(0, 0, 1)
		case "weekly":
			possibleTimeToCheck = possibleTimeToCheck.AddDate(0, 0, 7)
		default:
			log.Fatalf("unknown reapeat value of %v. Only daily and weekly are allowed\n", eventData.Repeat)
		}
	}
	return times
}

func createTitleMessage(eventTitle string, startTime time.Time) discord.WebhookWithComponent {
	return discord.WebhookWithComponent{
		WebhookParams: discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       eventTitle,
					Description: fmt.Sprintf("Event starts at %v", startTime.Format(time.RFC3339)),
				},
			},
		},
	}
}

func createGameMessage(gameTitle, gameDescription string) discord.WebhookWithComponent {
	return discord.WebhookWithComponent{
		WebhookParams: discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       gameTitle,
					Description: gameDescription,
				},
			},
		},
		Components: []discord.Component{
			{
				Type: 1,
				Components: []discord.Component{
					{
						Type:     2,
						Label:    "Add Me",
						Style:    3, // Green / Success Button
						CustomID: CustomIDButtonAddUserToGame,
					},
					{
						Type:     2,
						Label:    "Remove Me",
						Style:    4, // Red / Danger Button
						CustomID: CustomIDButtonRemoveUserFromGame,
					},
				},
			},
		},
	}
}
