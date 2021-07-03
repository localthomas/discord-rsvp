package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bsdlp/discord-interactions-go/interactions"
	"github.com/localthomas/discord-rsvp/discord"
)

const CustomIDButtonAddUserToGame = "add_user_to_game"
const CustomIDButtonRemoveUserFromEvent = "remove_user_from_event"

type InteractionHandler func(w http.ResponseWriter, interaction discord.ButtonInteraction, argument string)

type InteractionRouter struct {
	customIDHandlerMapping map[string]InteractionHandler
}

func NewInteractionRouter() InteractionRouter {
	return InteractionRouter{
		customIDHandlerMapping: make(map[string]InteractionHandler),
	}
}

func (i *InteractionRouter) RegisterHandler(customID string, handler InteractionHandler) {
	i.customIDHandlerMapping[customID] = handler
}

func (i *InteractionRouter) InteractionEndpoint(discordPubkey []byte) http.Handler {
	return discord.Verify(discordPubkey, http.HandlerFunc(i.interactionEndpointInternal))
}

func (i *InteractionRouter) interactionEndpointInternal(w http.ResponseWriter, r *http.Request) {
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
		i.interactionHandler(w, interaction)
	}
}

func (i *InteractionRouter) interactionHandler(w http.ResponseWriter, interaction discord.ButtonInteraction) {
	customID := interaction.DataInternal.CustomID
	// parse custom id as "command_with_underscores After the first whitespace, free text follows"
	splitted := strings.SplitN(customID, " ", 2)
	customID = splitted[0]
	argument := ""
	if len(splitted) > 1 {
		argument = splitted[1]
	}
	handler, ok := i.customIDHandlerMapping[customID]
	if !ok {
		fmt.Printf("unknown custom_id: %v\n", customID)
		return
	}
	handler(w, interaction, argument)
}

func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}
