package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const tokenURL = "https://discord.com/api/v8/oauth2/token"

type WebhookTokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    uint64 `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Webhook      struct {
		ApplicationID string `json:"application_id"`
		Name          string `json:"name"`
		URL           string `json:"url"`
		ChannelID     string `json:"channel_id"`
		Token         string `json:"token"`
		Type          int    `json:"type"`
		GuildID       string `json:"guild_id"`
		ID            string `json:"id"`
	} `json:"webhook"`
}

// GenerateWebhookOauthURL generates a URL for accessing a channel and the state for checking.
func GenerateWebhookOauthURL(clientID, redirectURI string) (string, string) {
	state := randStringBytes(16)
	escapedRedirectURI := url.QueryEscape(redirectURI)
	return fmt.Sprintf("https://discord.com/api/oauth2/authorize?response_type=code&client_id=%v&scope=webhook.incoming&state=%v&redirect_uri=%v", clientID, state, escapedRedirectURI), state
}

func RefreshToken(clientID, clientSecret, refreshToken string) (WebhookTokenResponse, error) {
	data := url.Values{}
	data.Add("client_id", clientID)
	data.Add("client_secret", clientSecret)
	data.Add("grant_type", "refresh_token")
	data.Add("refresh_token", refreshToken)
	return makeTokenRequest(tokenURL, data)
}

// Request a token, if the code is available
func RequestToken(clientID, clientSecret, code, redirectURI string) (WebhookTokenResponse, error) {
	data := url.Values{}
	data.Add("client_id", clientID)
	data.Add("client_secret", clientSecret)
	data.Add("code", code)
	data.Add("grant_type", "authorization_code")
	data.Add("redirect_uri", redirectURI)
	return makeTokenRequest(tokenURL, data)
}

func makeTokenRequest(apiUrl string, data url.Values) (WebhookTokenResponse, error) {
	u, _ := url.ParseRequestURI(apiUrl)
	urlStr := u.String()

	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	response, err := client.Do(r)
	if err != nil {
		return WebhookTokenResponse{}, fmt.Errorf("could not make POST request for token: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		// On error, read the response body, which contains more information
		builder := strings.Builder{}
		_, err := io.Copy(&builder, response.Body)
		if err != nil {
			return WebhookTokenResponse{}, fmt.Errorf("token POST request had not-ok status code %v, but the body could not be read", response.StatusCode)
		}
		return WebhookTokenResponse{}, fmt.Errorf("token POST request had not-ok status code %v: %v", response.StatusCode, builder.String())
	}

	responseToken := WebhookTokenResponse{}
	err = json.NewDecoder(response.Body).Decode(&responseToken)
	if err != nil {
		return WebhookTokenResponse{}, fmt.Errorf("received unknown JSON data: %w", err)
	}

	return responseToken, nil
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
