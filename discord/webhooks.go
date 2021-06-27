package discord

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type WebhookWithComponent struct {
	discordgo.WebhookParams
	// Components contains optional interactive components
	Components []Component `json:"components,omitempty"`
}

type Component struct {
	// Type defines the type of the component. Can be 1 (ActionRow) or 2 (Button)
	Type int `json:"type"`
	// Label is used to label the component
	Label string `json:"label,omitempty"`
	// Style defines the type of button, if type = 2.
	// Note that certain styles require other fields, see
	// https://discord.com/developers/docs/interactions/message-components#buttons-button-styles
	Style int `json:"style,omitempty"`
	// CustomID is a developer-defined identifier for the button, max 100 characters
	CustomID string `json:"custom_id,omitempty"`
	// Components contains an optional list of child components
	Components []Component `json:"components,omitempty"`
}

func SendWebhookWithComponents(session *discordgo.Session, webhookID, token string, wait bool, data WebhookWithComponent) (*discordgo.Message, error) {
	uri := discordgo.EndpointWebhookToken(webhookID, token)

	if wait {
		uri += "?wait=true"
	}

	response, err := session.RequestWithBucketID(http.MethodPost, uri, data, discordgo.EndpointWebhookToken("", ""))
	if !wait || err != nil {
		return nil, fmt.Errorf("could not make request: %w", err)
	}

	st := &discordgo.Message{}
	err = json.Unmarshal(response, &st)
	return st, err
}

func DeleteWebhookMessage(session *discordgo.Session, webhookID, token, messageID string) error {
	uri := discordgo.EndpointWebhookToken(webhookID, token) + "/messages/" + messageID

	_, err := session.RequestWithBucketID(http.MethodDelete, uri, nil, discordgo.EndpointWebhookToken("", ""))
	if err != nil {
		return fmt.Errorf("could not delete webhook message: %w", err)
	}
	return nil
}
