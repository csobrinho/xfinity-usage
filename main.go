package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	tokenURL  = "https://xerxes-sub.xerxessecure.com/xerxes-ctrl/oauth/token"
	usageURL  = "https://gw.api.dh.comcast.com/galileo/graphql"
	usageBody = `{"operationName":"InternetDataUsage","variables":{},"query":"query InternetDataUsage { accountByServiceAccountId { internet { usage { inPaidOverage courtesy { totalAllowableCourtesy usedCourtesy remainingCourtesy } monthlyUsage { policy month year startDate endDate daysRemaining currentUsage { value unit } allowableUsage { value unit } overage overageCharge maximumOverageCharge courtesyCredit } } } } }"}`
)

func init() {
	flag.DurationVar(&cfg.timeout, "timeout", 30*time.Second, "timeout in seconds")
	flag.StringVar(&cfg.clientID, "client_id", "xfinity-android-application", "OAuth client id")
	flag.StringVar(&cfg.mqttClientID, "mqtt_client_id", "xfinity-usage-go", "MQTT client id")
	flag.StringVar(&cfg.mqttStateTopic, "mqtt_state_topic", "homeassistant/sensor/xfinity_internet/state", "MQTT topic")

	flag.StringVar(&cfg.clientSecret, "client_secret", os.Getenv("CLIENT_SECRET"), "OAuth client secret")
	flag.StringVar(&cfg.refreshToken, "refresh_token", os.Getenv("REFRESH_TOKEN"), "OAuth refresh token")
	flag.StringVar(&cfg.applicationID, "application_id", os.Getenv("APPLICATION_ID"), "OAuth application id")
	flag.StringVar(&cfg.mqttURL, "mqtt_url", os.Getenv("MQTT_URL"), "MQTT url")
	flag.StringVar(&cfg.mqttUsername, "mqtt_username", os.Getenv("MQTT_USERNAME"), "MQTT username")
	flag.StringVar(&cfg.mqttPassword, "mqtt_password", os.Getenv("MQTT_PASSWORD"), "MQTT password")

	flag.Parse()
}

func run(ctx context.Context) error {
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	client := &http.Client{Timeout: cfg.timeout}

	token, err := tokenRequest(ctx, client, cfg.refreshToken, cfg.clientID, cfg.clientSecret, cfg.applicationID)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	log.Println("main: token Expiry:", token.ExpiresIn)

	u, err := internetDataUsageRequest(ctx, client, token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get internet usage: %w", err)
	}
	if u.Data == nil || u.Data.Account == nil || u.Data.Account.Internet == nil ||
		u.Data.Account.Internet.Usage == nil || len(u.Data.Account.Internet.Usage.MonthlyUsage) == 0 {
		return fmt.Errorf("failed to process internet usage")
	}

	cur, err := u.Data.Account.Internet.Usage.MonthlyUsage[0].CurrentUsage.GB()
	if err != nil {
		return fmt.Errorf("failed to get internet usage in gb: %w", err)
	}

	log.Printf("main: usage %7.2f GB", cur)
	if allowed, err := u.Data.Account.Internet.Usage.MonthlyUsage[0].AllowableUsage.GB(); err == nil {
		log.Printf("main: allowed %7.2f GB", allowed)
	}
	if err := mqttPublish(ctx, cfg.mqttURL, cfg.mqttUsername, cfg.mqttPassword, cfg.clientID, cfg.mqttStateTopic, cur); err != nil {
		return fmt.Errorf("failed to publish to mqtt: %w", err)
	}
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalf("main: %v", err)
	}
}
