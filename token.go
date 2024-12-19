package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

var (
	tokenExtraHeaders = map[string]string{
		"User-Agent": "Dalvik/2.1.0 (Linux; U; Android 14; SM-G991B Build/G991BXXUEGXJE",
	}
	tokenExtraValues = map[string]string{
		"active_x1_account_count": "true",
		"partner_id":              "comcast",
		"mso_partner_hint":        "true",
		"scope":                   "profile",
		"rm_hint":                 "true",
	}
)

func tokenRequest(ctx context.Context, client *http.Client, refreshToken, clientID, clientSecret string) (*oauth2.Token, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	for key, value := range tokenExtraValues {
		data.Set(key, value)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range tokenExtraHeaders {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, body)
	}

	// Parse the token response
	var token oauth2.Token
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &token, nil
}
