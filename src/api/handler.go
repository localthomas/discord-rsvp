package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/localthomas/discord-rsvp/discord"
)

func HandleAddUserToGame(w http.ResponseWriter, interaction discord.ButtonInteraction) {
	embed, err := extractEmbed(interaction)
	if err != nil {
		fmt.Printf("could not extract embed from interaction: %v\n", err)
		return
	}

	// add the user that pressed the button to the field with all users that were added
	userID := interaction.Member.User.ID
	users := stringToUserList(embed.Fields[0].Value)
	// check if user is already in the list
	alreadyExists := false
	for _, user := range users {
		if user == userID {
			alreadyExists = true
			break
		}
	}
	if !alreadyExists {
		users = append(users, userID)
	}
	embed.Fields[0].Value = userListToString(users)

	writeResponse(w, embed)
}

func HandleRemoveUserfromGame(w http.ResponseWriter, interaction discord.ButtonInteraction) {
	embed, err := extractEmbed(interaction)
	if err != nil {
		fmt.Printf("could not extract embed from interaction: %v\n", err)
		return
	}

	// remove the user that pressed the button from the field with all users that were added
	userID := interaction.Member.User.ID
	users := stringToUserList(embed.Fields[0].Value)
	for index, user := range users {
		if user == userID {
			if index+1 == len(users) {
				users = users[:index]
			} else {
				users = append(users[:index], users[index+1:]...)
			}
			break
		}
	}
	// special case: when the list of users is empty, just remove the field
	if len(users) == 0 {
		embed.Fields = nil
	} else {
		embed.Fields[0].Value = userListToString(users)
	}

	writeResponse(w, embed)
}

func extractEmbed(interaction discord.ButtonInteraction) (*discordgo.MessageEmbed, error) {
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
		return nil, fmt.Errorf("unknown state: message did not have embed")
	}
	return embed, nil
}

func writeResponse(w http.ResponseWriter, embed *discordgo.MessageEmbed) {
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

func userMention(userID string) string {
	return fmt.Sprintf("<@%v>", userID)
}

const userListSplitValue = "\n"

// stringToUserList converts the given string to a list of user IDs.
// Use userListToString for the reverse operation.
func stringToUserList(value string) []string {
	// userID mention format: <@12345>
	usersRaw := strings.Split(value, userListSplitValue)
	userIDs := make([]string, 0)
	for i := range usersRaw {
		userID := strings.Trim(usersRaw[i], "<>")
		userID = strings.TrimPrefix(userID, "@")
		// Note: skip empty user IDs
		if userID != "" {
			userIDs = append(userIDs, userID)
		}
	}
	return userIDs
}

// userListToString converts the given ids to a single string of user mentions.
// Use stringToUserList for the reverse operation.
func userListToString(userIDs []string) string {
	userMentions := make([]string, 0)
	for _, userID := range userIDs {
		// Note: skip empty user IDs
		if userID != "" {
			userMentions = append(userMentions, userMention(userID))
		}
	}
	return strings.Join(userMentions, userListSplitValue)
}
