package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func mqttPublish(ctx context.Context, mqttURL, mqttUsername, mqttPassword, mqttClientID, mqttStateTopic string, usage float32) error {
	// TODO: Make sure the attributes and config topics also exist.

	u, err := url.Parse(mqttURL)
	if err != nil {
		return fmt.Errorf("failed to parse mqtt server url: %v", err)
	}
	logger := log.New(os.Stdout, "mqtt: ", log.Default().Flags()|log.Lmsgprefix)
	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		SessionExpiryInterval:         10,
		ConnectUsername:               mqttUsername,
		ConnectPassword:               []byte(mqttPassword),
		Debug:                         logger,
		Errors:                        logger,
		PahoDebug:                     logger,
		PahoErrors:                    logger,
		ClientConfig: paho.ClientConfig{
			ClientID:      mqttClientID,
			OnClientError: func(err error) { log.Printf("mqtt: client error: %s", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					log.Printf("mqtt: server requested disconnect: %s", d.Properties.ReasonString)
				} else {
					log.Printf("mqtt: server requested disconnect; reason code: %d", d.ReasonCode)
				}
			},
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
	_, err = c.Publish(ctx, &paho.Publish{
		Topic:   mqttStateTopic,
		Retain:  true,
		QoS:     1,
		Payload: []byte(fmt.Sprintf("%.2f", usage)),
	})
	return err
}
