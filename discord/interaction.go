package discord

import (
	"crypto/ed25519"
	"net/http"

	"github.com/bsdlp/discord-interactions-go/interactions"
)

/*
type Interaction struct {
	Version int     `json:"version"`
	Type    int     `json:"type"`
	Token   string  `json:"token"`
	Message Webhook `json:"message"`
	ID      string  `json:"id"`
	Data    Data    `json:"data"`
}

type Data struct {
	CustomID string `json:"custom_id"`
}
*/

// Verify implements the Security and Authorization section of the Discord API.
// https://discord.com/developers/docs/interactions/slash-commands#security-and-authorization
func Verify(publicKey []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		verified := interactions.Verify(r, ed25519.PublicKey(publicKey))
		if !verified {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
