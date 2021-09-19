package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/localthomas/discord-rsvp/discord"
)

func HandleAddUserToGame(w http.ResponseWriter, interaction discord.ButtonInteraction, argument string) {
	embed := &discordgo.MessageEmbed{}
	if len(interaction.Message.Embeds) > 1 {
		embed = interaction.Message.Embeds[1]
	} else {
		interaction.Message.Embeds = append(interaction.Message.Embeds, embed)
	}

	embed.Title = "Attendees"
	embed.Color = 0x3ba55d

	// add the user that pressed the button to the embed with all users that were added to a game
	userID := interaction.Member.User.ID
	var field *discordgo.MessageEmbedField
	// find the field for the game
	for _, fieldToTest := range embed.Fields {
		if strings.HasPrefix(fieldToTest.Name, argument) {
			field = fieldToTest
			break
		}
	}
	if field == nil {
		// create a new field with default values
		field = &discordgo.MessageEmbedField{
			Name:   argument,
			Inline: true,
		}
		embed.Fields = append(embed.Fields, field)
	}

	users := stringToUserList(field.Value)
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
	field.Value = userListToString(users)
	// set the field title to "Game (2)", where 2 is the number of users (attendees)
	field.Name = argument + fmt.Sprintf(" (%v)", len(users))

	writeResponse(w, interaction.Message)
}

func HandleRemoveUserFromEvent(w http.ResponseWriter, interaction discord.ButtonInteraction, argument string) {
	var embed *discordgo.MessageEmbed
	if len(interaction.Message.Embeds) > 1 {
		embed = interaction.Message.Embeds[1]
	} else {
		// do nothing, if no embed for the attendees was found
		writeResponse(w, interaction.Message)
		return
	}

	// remove the user that pressed the button from all the fields
	userID := interaction.Member.User.ID
	for i := 0; i < len(embed.Fields); i++ {
		users := stringToUserList(embed.Fields[i].Value)
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
			// remove routine
			if i+1 == len(embed.Fields) {
				embed.Fields = embed.Fields[:i]
			} else {
				embed.Fields = append(embed.Fields[:i], embed.Fields[i+1:]...)
			}
			i-- // since one field was removed, the list length is now -1 and the next element got a new index: i-1
		} else {
			embed.Fields[i].Value = userListToString(users)
			// set the field title to "Game (2)", where 2 is the number of users (attendees)
			embed.Fields[i].Name = extractGameNameFromFieldName(embed.Fields[i].Name) + fmt.Sprintf(" (%v)", len(users))
		}
	}

	// special case: if no fields are on the embed, remove it
	if len(embed.Fields) == 0 {
		interaction.Message.Embeds = interaction.Message.Embeds[:1]
	}

	writeResponse(w, interaction.Message)
}

func extractGameNameFromFieldName(fieldName string) string {
	expression := regexp.MustCompile(`(.*) \([0-9]+\)$`)
	matches := expression.FindStringSubmatch(fieldName)
	if len(matches) >= 2 {
		return matches[1]
	} else {
		return ""
	}
}

func extractEmbed(interaction discord.ButtonInteraction) (*discordgo.MessageEmbed, error) {
	// extract the current embed from the interaction
	var embed *discordgo.MessageEmbed
	if len(interaction.Message.Embeds) > 1 {
		embed = interaction.Message.Embeds[1]
	} else {
		embed = &discordgo.MessageEmbed{}
	}
	return embed, nil
}

func writeResponse(w http.ResponseWriter, message discord.WebhookWithComponent) {
	response := discord.ButtonInteractionResponse{
		Type: 7,
		Data: message,
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
