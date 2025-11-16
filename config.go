package main

import (
	"fmt"
	"time"
)

type config struct {
	timeout            time.Duration
	clientID           string
	clientSecret       string
	refreshToken       string
	accessToken        string
	applicationID      string
	mqttURL            string
	mqttClientID       string
	mqttStateTopic     string
	mqttUsername       string
	mqttPassword       string
	prometheusEndpoint string
	prometheusJob      string
	query              string
}

var cfg config

func (c config) validate() error {
	if c.clientID == "" {
		return fmt.Errorf("missing --client_id")
	}
	if c.clientSecret == "" {
		return fmt.Errorf("missing --client_secret")
	}
	if c.refreshToken == "" && c.accessToken == "" {
		return fmt.Errorf("either --refresh_token or --access_token must be provided")
	}
	if c.mqttURL == "" {
		return fmt.Errorf("missing --mqtt_url")
	}
	if c.mqttClientID == "" {
		return fmt.Errorf("missing --mqtt_client_id")
	}
	if c.mqttStateTopic == "" {
		return fmt.Errorf("missing --mqtt_state_topic")
	}
	if c.mqttUsername == "" {
		return fmt.Errorf("missing --mqtt_username")
	}
	if c.mqttPassword == "" {
		return fmt.Errorf("missing --mqtt_password")
	}
	return nil
}
