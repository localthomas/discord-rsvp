package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bsdlp/discord-interactions-go/interactions"
	"github.com/bwmarrin/discordgo"
	"github.com/localthomas/discord-rsvp/discord"
)

const Version = 1 // static version of this software
const WebhookTokenEndpoint = "/webhook-token"
const ConfigFilePath = "config/config.json"

const CustomIDButtonAddUserToGame = "add_user_to_game"
const CustomIDButtonRemoveUserFromGame = "remove_user_from_game"

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

	http.Handle("/", discord.Verify(discordPubkey, interactionEndpoint(interactionHandler)))
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
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func interactionEndpoint(next func(w http.ResponseWriter, interaction discord.ButtonInteraction)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var interaction discord.ButtonInteraction
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
				fmt.Printf("error on sending pong as HTTP-Response: %v\n", err)
			}
		} else {
			next(w, interaction)
		}
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

func interactionHandler(w http.ResponseWriter, interaction discord.ButtonInteraction) {
	if interaction.DataInternal.CustomID == CustomIDButtonAddUserToGame {
		// extract the current embed from the interaction
		var embed *discordgo.MessageEmbed
		if len(interaction.Message.Embeds) > 0 {
			embed = interaction.Message.Embeds[0]
			if len(embed.Fields) == 0 {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   "Accepted",
					Value:  "",
					Inline: true,
				})
			}
		} else {
			fmt.Println("unknown state: message did not have embed")
			return
		}

		// add the user that pressed the button to the field with all users that were added
		userID := interaction.Member.User.ID
		const SplitValue = "\n"
		users := strings.Split(embed.Fields[0].Value, SplitValue)
		// check if user is already in the list
		alreadyExists := false
		for _, user := range users {
			if user == userMention(userID) {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			users = append(users, userMention(userID))
		}
		embed.Fields[0].Value = strings.Join(users, SplitValue)

		response := discord.ButtonInteractionResponse{
			Type: 7,
			Data: discord.WebhookWithComponent{
				WebhookParams: discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{
						embed,
					},
				},
			},
		}
		err := writeJSON(w, response)
		if err != nil {
			fmt.Printf("could not write interaction response: %v\n", err)
		}
	}
}

func userMention(userID string) string {
	return fmt.Sprintf("<@%v>", userID)
}
