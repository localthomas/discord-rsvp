package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Webhook is the entry-structure for Discord Webhooks.
// More information at https://discord.com/developers/docs/resources/webhook#execute-webhook
type Webhook struct {
	// TTS is text-to-speech
	TTS bool `json:"tts,omitempty"`
	// Content the message contents (up to 2000 characters)
	Content string `json:"content,omitempty"`
	// Username override the default username of the webhook
	Username string `json:"username,omitempty"`
	// AvatarURL override the default avatar of the webhook
	AvatarURL string `json:"avatar_url,omitempty"`
	// Embeds contains embeddings for the message
	Embeds []WebhookEmbed `json:"embeds,omitempty"`
	// Components contains optional interactive components
	Components []Component `json:"components,omitempty"`
	// The ID of the message, read-only
	ID string `json:"id"`
}

type WebhookEmbed struct {
	Author      WebhookEmbedAuthor  `json:"author,omitempty"`
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []WebhookEmbedField `json:"fields,omitempty"`
	Footer      WebhookEmbedFooter  `json:"footer,omitempty"`
}

type WebhookEmbedAuthor struct {
	Name string `json:"name,omitempty"`
}

type WebhookEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type WebhookEmbedFooter struct {
	Text string `json:"text,omitempty"`
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

// WebhookClient holds all information for executing webhooks
type WebhookClient struct {
	webhookID    string
	webhookToken string
	authString   string
	thisURL      string
	version      int
}

func NewWebhookClient(webhookID, webhookToken, authString, thisInstanceURL string, version int) WebhookClient {
	return WebhookClient{
		webhookID:    webhookID,
		webhookToken: webhookToken,
		authString:   authString,
		thisURL:      thisInstanceURL,
		version:      version,
	}
}

// SendWebhook sends the given message to the URL.
func (w WebhookClient) SendWebhook(message Webhook) error {
	requestBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("could not marshal message: %w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		generateWebhookURL(w.webhookID, w.webhookToken)+"?wait=true",
		bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("could not create POST request: %w", err)
	}
	request.Header.Set("Authorization", w.authString)
	request.Header.Set("User-Agent", fmt.Sprintf("DiscordBot (%v, %v)", w.thisURL, w.version))
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("could not make POST request for webhook: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		// On error, read the response body, which contains more information
		builder := strings.Builder{}
		_, err := io.Copy(&builder, response.Body)
		if err != nil {
			return fmt.Errorf("webhook POST request had not-ok status code %v, but the body could not be read", response.StatusCode)
		}
		return fmt.Errorf("webhook POST request had not-ok status code %v: %v", response.StatusCode, builder.String())
	}

	return nil
}

func generateWebhookURL(id, token string) string {
	return fmt.Sprintf("https://discord.com/api/webhooks/%v/%v", id, token)
}
