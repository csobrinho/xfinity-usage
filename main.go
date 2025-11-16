package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	tokenURL  = "https://xerxes-sub.xerxessecure.com/xerxes-ctrl/oauth/token"
	usageURL  = "https://gw.api.dh.comcast.com/galileo/graphql"
	usageBody = `{"operationName":"InternetDataUsage","variables":{},"query":"query InternetDataUsage { accountByServiceAccountId { internet { usage { inPaidOverage courtesy { totalAllowableCourtesy usedCourtesy remainingCourtesy } monthlyUsage { policy month year startDate endDate daysRemaining currentUsage { value unit } allowableUsage { value unit } overage overageCharge maximumOverageCharge courtesyCredit } } } } }"}`
)

func init() {
	flag.DurationVar(&cfg.timeout, "timeout", 90*time.Second, "timeout in seconds")
	flag.StringVar(&cfg.clientID, "client_id", "xfinity-android-application", "OAuth client id")
	flag.StringVar(&cfg.mqttClientID, "mqtt_client_id", "xfinity-usage-go", "MQTT client id")
	flag.StringVar(&cfg.mqttStateTopic, "mqtt_state_topic", "homeassistant/sensor/xfinity_internet/state", "MQTT topic")

	flag.StringVar(&cfg.clientSecret, "client_secret", os.Getenv("CLIENT_SECRET"), "OAuth client secret")
	flag.StringVar(&cfg.refreshToken, "refresh_token", os.Getenv("REFRESH_TOKEN"), "OAuth refresh token")
	flag.StringVar(&cfg.accessToken, "access_token", os.Getenv("ACCESS_TOKEN"), "OAuth access token")
	flag.StringVar(&cfg.applicationID, "application_id", os.Getenv("APPLICATION_ID"), "OAuth application id")
	flag.StringVar(&cfg.mqttURL, "mqtt_url", os.Getenv("MQTT_URL"), "MQTT url")
	flag.StringVar(&cfg.mqttUsername, "mqtt_username", os.Getenv("MQTT_USERNAME"), "MQTT username")
	flag.StringVar(&cfg.mqttPassword, "mqtt_password", os.Getenv("MQTT_PASSWORD"), "MQTT password")
	flag.StringVar(&cfg.prometheusJob, "prometheus_job", "xfinity-usage", "Prometheus job name")
	flag.StringVar(&cfg.prometheusEndpoint, "prometheus_endpoint", os.Getenv("PROMETHEUS_ENDPOINT"), "Prometheus Pushgateway endpoint")
	flag.StringVar(&cfg.query, "query", os.Getenv("QUERY"), "GraphQL query to test")

	flag.Parse()
}

func retryPolicyWithMetrics(ctx context.Context, resp *http.Response, err error) (bool, error) {
	shouldRetry, retryErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	if shouldRetry {
		host := "unknown"
		method := "unknown"
		statusCode := 0
		if resp != nil {
			if resp.Request != nil {
				host = resp.Request.URL.Host
				method = resp.Request.Method
			}
			statusCode = resp.StatusCode
		}
		recordRetry(host, method, statusCode)
	}
	return shouldRetry, retryErr
}

func actionRunQuery(ctx context.Context, client *retryablehttp.Client, accessToken, graphql string) error {
	body, err := query(ctx, client, accessToken, usageURL, "POST", strings.NewReader(graphql), usageExtraHeaders)
	if err != nil {
		return err
	}
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}
	log.Println(string(pretty))

	return nil
}

func getAccessToken(ctx context.Context, client *retryablehttp.Client) (string, error) {
	// Short-circuit if access token is already provided.
	if cfg.accessToken != "" {
		log.Println("main: using provided access token")
		return cfg.accessToken, nil
	}

	// Refresh OAuth token.
	tokenStart := time.Now()
	token, err := tokenRequest(ctx, client, cfg.refreshToken, cfg.clientID, cfg.clientSecret, cfg.applicationID)
	tokenRefreshDuration.Observe(time.Since(tokenStart).Seconds())
	if err != nil {
		recordError(errorCategoryTokenRefresh)
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}
	log.Println("main: token Expiry:", token.ExpiresIn)
	return token.AccessToken, nil
}

func actionFetchUsageData(ctx context.Context, client *retryablehttp.Client, accessToken string) error {
	usageStart := time.Now()
	u, err := internetDataUsageRequest(ctx, client, accessToken)
	usageFetchDuration.Observe(time.Since(usageStart).Seconds())
	if err != nil {
		recordError(errorCategoryUsageFetch)
		return fmt.Errorf("failed to get internet usage: %w", err)
	}

	// Parse and validate usage data.
	if u.Data == nil || u.Data.Account == nil || u.Data.Account.Internet == nil ||
		u.Data.Account.Internet.Usage == nil || len(u.Data.Account.Internet.Usage.MonthlyUsage) == 0 {
		recordError(errorCategoryUsageParse)
		return fmt.Errorf("failed to process internet usage")
	}

	monthlyUsage := u.Data.Account.Internet.Usage.MonthlyUsage[0]
	cur, err := monthlyUsage.CurrentUsage.GB()
	if err != nil {
		recordError(errorCategoryUsageParse)
		return fmt.Errorf("failed to get internet usage in gb: %w", err)
	}

	log.Printf("main: usage %7.2f GB", cur)
	if allowed, err := monthlyUsage.AllowableUsage.GB(); err == nil {
		log.Printf("main: allowed %7.2f GB", allowed)
	}

	// Publish to MQTT.
	mqttStart := time.Now()
	if err := mqttPublish(ctx, cfg.mqttURL, cfg.mqttUsername, cfg.mqttPassword, cfg.clientID, cfg.mqttStateTopic, cur); err != nil {
		mqttPublishDuration.Observe(time.Since(mqttStart).Seconds())
		recordError(errorCategoryMQTTPublish)
		return fmt.Errorf("failed to publish to mqtt: %w", err)
	}
	mqttPublishDuration.Observe(time.Since(mqttStart).Seconds())

	// Record success metrics.
	recordSuccess()
	return nil
}

func run(ctx context.Context) error {
	// Increment total runs counter.
	runsTotal.Inc()

	// Validate configuration.
	if err := cfg.validate(); err != nil {
		recordError(errorCategoryConfigValidation)
		return fmt.Errorf("failed to validate config: %w", err)
	}

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.CheckRetry = retryPolicyWithMetrics

	// Get access token (either from config or refresh).
	accessToken, err := getAccessToken(ctx, client)
	if err != nil {
		return err
	}

	if cfg.query != "" {
		log.Println("main: running test query")
		return actionRunQuery(ctx, client, accessToken, cfg.query)
	}
	return actionFetchUsageData(ctx, client, accessToken)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	setBuildInfo(version, runtime.Version())
	recordRunStart()

	// Track execution duration.
	start := time.Now()
	defer func() { executionDuration.Observe(time.Since(start).Seconds()) }()

	err := run(ctx)
	if err != nil {
		recordFailure()
	}

	if cfg.prometheusEndpoint != "" {
		if perr := pushMetrics(ctx, cfg.prometheusEndpoint, cfg.prometheusJob); perr != nil {
			log.Printf("main: failed to push metrics: %v", perr)
			recordError(errorCategoryMetricsPush)
		} else {
			log.Println("main: metrics pushed successfully")
		}
	}

	if err != nil {
		log.Fatalf("main: %v", err)
	}
	log.Println("main: all done ✅")
}
