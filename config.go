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
	applicationID      string
	mqttURL            string
	mqttClientID       string
	mqttStateTopic     string
	mqttUsername       string
	mqttPassword       string
	prometheusEndpoint string
	prometheusJob      string
}

var cfg config

func (c config) validate() error {
	if c.clientID == "" {
		return fmt.Errorf("missing --client_id")
	}
	if c.clientSecret == "" {
		return fmt.Errorf("missing --client_secret")
	}
	if c.refreshToken == "" {
		return fmt.Errorf("missing --refresh_token")
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
