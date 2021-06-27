package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/localthomas/discord-rsvp/api"
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
		var session *discordgo.Session

		// never ending loop that executes tasks
		for {
			// check if the token needs to be refreshed
			if time.Until(state.ExpiresAt) < 1*time.Hour && state.RefreshToken != "" {
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

			if session == nil && state.AuthorizationToken != "" {
				newSession, err := discordgo.New(state.AuthorizationTokenType + " " + state.AuthorizationToken)
				if err != nil {
					log.Fatalf("could not create session: %v\n", err)
				} else {
					session = newSession
					session.UserAgent = fmt.Sprintf("DiscordBot (%v, %v)", config.ThisInstanceURL, Version)
				}
			}

			if session != nil {
				handleEventScheduling(session, &state, config)
			}

			time.Sleep(1 * time.Second)
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

	handlerRouter := api.NewInteractionRouter()
	handlerRouter.RegisterHandler(api.CustomIDButtonAddUserToGame, api.HandleAddUserToGame)
	handlerRouter.RegisterHandler(api.CustomIDButtonRemoveUserFromGame, api.HandleRemoveUserfromGame)

	http.Handle("/", handlerRouter.InteractionEndpoint(discordPubkey))
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
					return
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
	log.Fatal(http.ListenAndServe(":80", nil))
}
