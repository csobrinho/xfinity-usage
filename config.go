package main

import (
	"fmt"
	"time"
)

type config struct {
	timeout             time.Duration
	verbose             int
	clientID            string
	clientSecret        string
	refreshToken        string
	accessToken         string
	idToken             string
	applicationID       string
	mqttURL             string
	mqttClientID        string
	mqttStateTopic      string
	mqttAttributesTopic string
	mqttUsername        string
	mqttPassword        string
	prometheusEndpoint  string
	prometheusJob       string
	query               string
}

var cfg config

func (c config) validate() error {
	if c.clientID == "" {
		return fmt.Errorf("missing --client_id")
	}
	if c.refreshToken == "" && c.accessToken == "" {
		return fmt.Errorf("either --refresh_token or --access_token must be provided")
	}
	if c.accessToken != "" && c.idToken == "" {
		return fmt.Errorf("if --access_token is provided, --id_token must also be provided")
	}
	if c.accessToken == "" && c.clientSecret == "" {
		return fmt.Errorf("missing --client_secret")
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
	if c.mqttAttributesTopic == "" {
		return fmt.Errorf("missing --mqtt_attributes_topic")
	}
	if c.mqttUsername == "" {
		return fmt.Errorf("missing --mqtt_username")
	}
	if c.mqttPassword == "" {
		return fmt.Errorf("missing --mqtt_password")
	}
	return nil
}
