package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func mqttPublish(ctx context.Context, mqttURL, mqttUsername, mqttPassword, mqttClientID, mqttStateTopic, mqttAttributesTopic string, usage float32, attributes *UsageAttributes) error {
	u, err := url.Parse(mqttURL)
	if err != nil {
		return fmt.Errorf("failed to parse mqtt server url: %v", err)
	}
	mqttLogger := &logger{prefix: "mqtt: "}
	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		SessionExpiryInterval:         10,
		ConnectUsername:               mqttUsername,
		ConnectPassword:               []byte(mqttPassword),
		Debug:                         mqttLogger,
		Errors:                        mqttLogger,
		PahoDebug:                     mqttLogger,
		PahoErrors:                    mqttLogger,
	}
	cfg.ClientConfig = paho.ClientConfig{
		ClientID: mqttClientID,
		OnClientError: func(err error) {
			cfg.Errors.Printf("client error: %s", err)
		},
		OnServerDisconnect: func(d *paho.Disconnect) {
			if d.Properties != nil {
				cfg.Errors.Printf("server requested disconnect: %s", d.Properties.ReasonString)
			} else {
				cfg.Errors.Printf("server requested disconnect; reason code: %d", d.ReasonCode)
			}
		},
	}
	c, err := autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		c.Disconnect(ctx)
		<-c.Done()
	}()

	if err = c.AwaitConnection(ctx); err != nil {
		return err
	}

	// Publish state (numeric value).
	if _, err = c.Publish(ctx, &paho.Publish{
		Topic:   mqttStateTopic,
		Retain:  true,
		QoS:     1,
		Payload: fmt.Appendf(nil, "%.2f", usage),
	}); err != nil {
		return fmt.Errorf("failed to publish state: %w", err)
	}

	// Publish attributes (JSON).
	attrs, err := json.Marshal(attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}
	if _, err = c.Publish(ctx, &paho.Publish{
		Topic:   mqttAttributesTopic,
		Retain:  true,
		QoS:     1,
		Payload: attrs,
	}); err != nil {
		return fmt.Errorf("failed to publish attributes: %w", err)
	}

	return nil
}
