package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bsdlp/discord-interactions-go/interactions"
	"github.com/localthomas/discord-rsvp/discord"
)

const Version = 1 // static version of this software
const WebhookTokenEndpoint = "/webhook-token"
const ConfigFilePath = "config/config.json"

func main() {
	config, err := ReadConfig(ConfigFilePath)
	if err != nil {
		log.Fatalf("could not read configuration from file %v: %v", ConfigFilePath, err)
	}
	discordPubkey, err := hex.DecodeString(config.HexEncodedDiscordPublicKey)
	if err != nil {
		log.Fatalf("could not decode public key: %v", err)
	}

	state := ResumeState()
	// remove any expired token data
	if state.ExpiresAt.Before(time.Now()) {
		state.SetToken("", "", time.Time{}, "")
	}

	go func() {
		tmpSend := false
		// never ending loop that executes tasks
		for {
			time.Sleep(10 * time.Second)

			// check if the token needs to be refreshed
			if time.Until(state.ExpiresAt) < 1*time.Hour {
				token, err := discord.RefreshToken(
					config.ClientID,
					config.ClientSecret,
					state.RefreshToken)
				if err != nil {
					fmt.Printf("could not refresh access token: %v\n", err)
				} else {
					state.SetToken(
						token.TokenType,
						token.AccessToken,
						time.Now().Add(time.Duration(token.ExpiresIn)*time.Second),
						token.RefreshToken)
					fmt.Println("token was refreshed")
				}
			}

			if !tmpSend && state.AuthorizationToken != "" {
				client := discord.NewWebhookClient(
					state.WebhookID,
					state.WebhookToken,
					state.AuthorizationTokenType+" "+state.AuthorizationToken,
					config.ThisInstanceURL,
					Version,
				)

				message := discord.Webhook{
					Embeds: []discord.WebhookEmbed{
						{
							Title:       "Title!",
							Description: "Description",
						},
					},
					Components: []discord.Component{
						{
							Type: 1,
							Components: []discord.Component{
								{
									Type:     2,
									Label:    "Test",
									Style:    3,
									CustomID: "Test",
								},
							},
						},
					},
				}
				err = client.SendWebhook(message)
				if err != nil {
					log.Fatal(err)
				}
				tmpSend = true
			}
		}
	}()

	// print oauth-URL if no token is saved
	check := ""
	if state.AuthorizationToken == "" {
		accessURL, newCheck := discord.GenerateWebhookOauthURL(
			config.ClientID,
			config.ThisInstanceURL+WebhookTokenEndpoint)
		check = newCheck
		fmt.Println(accessURL)
	}

	http.Handle("/", discord.Verify(discordPubkey, http.HandlerFunc(interactionEndpoint)))
	http.Handle(WebhookTokenEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if stateStr := query.Get("state"); stateStr == check {
			if code := query.Get("code"); code != "" {
				token, err := discord.RequestToken(
					config.ClientID,
					config.ClientSecret,
					code,
					config.ThisInstanceURL+WebhookTokenEndpoint)
				if err != nil {
					fmt.Printf("could not request token: %v\n", err)
				}
				state.SetToken(
					token.TokenType,
					token.AccessToken,
					time.Now().Add(time.Duration(token.ExpiresIn)*time.Second),
					token.RefreshToken)
				state.SetWebhook(token.Webhook.ID, token.Webhook.Token)
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized!"))
			return
		}
		w.Write([]byte("Success!"))
	}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func interactionEndpoint(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var interaction interactions.Data
	err := json.NewDecoder(r.Body).Decode(&interaction)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("received unknown JSON data: %v\n", err)
		return
	}
	if interaction.Type == 1 {
		err = writeJSON(w, interactions.Data{
			Type: 1,
		})
		if err != nil {
			fmt.Printf("error on sending JSON to HTTP-Response: %v\n", err)
		}
	} else {
		fmt.Println("received unknown interaction of type", interaction.Type)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}
