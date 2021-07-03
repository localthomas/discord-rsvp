package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/localthomas/discord-rsvp/api"
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
				if alreadyCreated.Title == eventTitle && alreadyCreated.StartsAt.Equal(eventTime) {
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
			err := discord.DeleteWebhookMessage(session, event.WebhookID, event.WebhookToken, event.MessageID)
			if err != nil {
				fmt.Printf("could not delete message of event %v: %v\n", event.Title, err)
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
	message := createEventMessage(eventTitle, startTime, config.Games)
	messageReturn, err := discord.SendWebhookWithComponents(
		session,
		webhookID,
		webhookToken,
		true,
		message,
	)
	if err != nil {
		return fmt.Errorf("could not send webhook message: %w", err)
	}
	messageID := messageReturn.ID

	state.AddRsvpEvent(RsvpEvent{
		MessageID:    messageID,
		Title:        eventTitle,
		StartsAt:     startTime,
		WebhookID:    webhookID,
		WebhookToken: webhookToken,
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
loop:
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
			break loop
		}

		switch eventData.Repeat {
		case "daily":
			possibleTimeToCheck = possibleTimeToCheck.AddDate(0, 0, 1)
		case "weekly":
			possibleTimeToCheck = possibleTimeToCheck.AddDate(0, 0, 7)
		case "never":
			break loop
		default:
			log.Fatalf("unknown reapeat value of %v. Only daily, weekly and never are allowed\n", eventData.Repeat)
		}
	}
	return times
}

func createEventMessage(eventTitle string, startTime time.Time, games map[string]string) discord.WebhookWithComponent {
	// create a list of game names and descriptions and sort them
	gamesList := gamesToList(games)

	// prepare buttons for each game
	// Note: maximum amount of buttons in one actionRow is 5
	buttons := []discord.Component{}
	counter := 0
	tmpButtons := []discord.Component{}
	for _, game := range gamesList {
		tmpButtons = append(tmpButtons, discord.Component{
			Type:     2,
			Label:    game.Title,
			Style:    3, // Green / Success Button
			CustomID: api.CustomIDButtonAddUserToGame + " " + game.Title,
		})
		if counter < 4 {
			counter++
		} else {
			counter = 0
			buttons = append(buttons, discord.Component{
				Type:       1,
				Components: tmpButtons,
			})
			tmpButtons = []discord.Component{}
		}
	}
	// if tmpButtons still contains elements, add them to the end
	if len(tmpButtons) > 0 {
		buttons = append(buttons, discord.Component{
			Type:       1,
			Components: tmpButtons,
		})
	}
	// add remove button last
	buttons = append(buttons, discord.Component{
		Type: 1,
		Components: []discord.Component{
			{
				Type:     2,
				Label:    "Remove Me",
				Style:    4, // Red / Danger Button
				CustomID: api.CustomIDButtonRemoveUserFromEvent,
			},
		},
	})

	// prepare info fields for each game
	fields := []*discordgo.MessageEmbedField{}
	for _, game := range gamesList {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   game.Title,
			Value:  game.Description,
			Inline: true,
		})
	}

	return discord.WebhookWithComponent{
		WebhookParams: discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       eventTitle,
					Description: fmt.Sprintf("Event starts at %v.\nSelect the games you want to play via the buttons below.", startTime.Format(time.RFC1123)),
					Color:       0x01579b,
					Fields:      fields,
				},
			},
		},
		Components: buttons,
	}
}

type gameEntry struct {
	Title       string
	Description string
}

// gameEntrySorter joins a By function and a slice of gameEntries to be sorted.
type gameEntrySorter struct {
	games []gameEntry
	by    func(p1, p2 *gameEntry) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (g *gameEntrySorter) Len() int {
	return len(g.games)
}

// Swap is part of sort.Interface.
func (g *gameEntrySorter) Swap(i, j int) {
	g.games[i], g.games[j] = g.games[j], g.games[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (g *gameEntrySorter) Less(i, j int) bool {
	return g.by(&g.games[i], &g.games[j])
}

func gamesToList(games map[string]string) []gameEntry {
	list := []gameEntry{}
	for title, description := range games {
		list = append(list, gameEntry{
			Title:       title,
			Description: description,
		})
	}

	// sort by title
	sorter := &gameEntrySorter{
		games: list,
		by: func(p1, p2 *gameEntry) bool {
			return p1.Title < p2.Title
		},
	}
	sort.Sort(sorter)

	return list
}
