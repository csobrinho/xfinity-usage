# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that fetches Xfinity internet usage data and publishes it to MQTT. It's designed to run as a Kubernetes CronJob but can run standalone. The application authenticates via OAuth2 using a refresh token, queries a GraphQL API for usage data, and publishes the current usage in GB to an MQTT topic.

## Architecture

The codebase is a single-package Go application with a clear separation of concerns:

- **main.go**: Entry point and orchestration. Validates config, refreshes OAuth token, fetches usage data, publishes to MQTT, and pushes metrics.
- **config.go**: Configuration management via flags and environment variables. All config is stored in a single global `cfg` variable.
- **token.go**: OAuth2 token refresh logic. Makes HTTP POST to Xfinity's OAuth endpoint with required headers and parameters.
- **usage.go**: GraphQL query execution to fetch internet usage data. Contains `Usage` structs that parse the nested GraphQL response and a `GB()` method to normalize usage values from MB/GB/TB to GB.
- **mqtt.go**: MQTT client setup and message publishing using eclipse/paho.golang with autopaho for connection management.
- **metrics.go**: Prometheus metrics definitions and push logic. Tracks errors by category, success metrics, execution duration, and usage data.

The application flow is linear:
1. Parse config from flags/env vars
2. Refresh OAuth token using refresh token
3. Fetch usage data using access token
4. Extract current usage in GB
5. Publish to MQTT topic
6. Push metrics to Prometheus Pushgateway (if configured)
7. Exit

## Building and Running

Build the application:
```bash
go build -o xfinity-usage
```

Build with version information (recommended):
```bash
# Using git tag
VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X main.version=${VERSION}" -o xfinity-usage

# Or using git commit
COMMIT=$(git rev-parse --short HEAD)
go build -ldflags "-X main.version=${COMMIT}" -o xfinity-usage

# Or combine both
VERSION=$(git describe --tags --always --dirty)-$(git rev-parse --short HEAD)
go build -ldflags "-X main.version=${VERSION}" -o xfinity-usage
```

Run with environment variables:
```bash
export CLIENT_SECRET="your-client-secret"
export REFRESH_TOKEN="your-refresh-token"
export MQTT_URL="mqtt://mosquitto:1883"
export MQTT_USERNAME="user"
export MQTT_PASSWORD="pass"

./xfinity-usage
```

Or run with flags:
```bash
./xfinity-usage \
  -client_secret="your-client-secret" \
  -refresh_token="your-refresh-token" \
  -mqtt_url="mqtt://mosquitto:1883" \
  -mqtt_username="user" \
  -mqtt_password="pass"
```

Build Docker image:
```bash
docker build -t xfinity-usage .
```

Build Docker image with version:
```bash
VERSION=$(git describe --tags --always --dirty)
docker build --build-arg VERSION=${VERSION} -t xfinity-usage:${VERSION} .
```

## Testing

This project does not currently have automated tests. Manual testing requires:
1. Valid `CLIENT_SECRET` and `REFRESH_TOKEN` from an Android emulator
2. Running MQTT broker
3. Valid Xfinity account with internet service

## Key Implementation Details

**Authentication**: The refresh token must be extracted from an Android emulator (see README.md for extraction process).

**HTTP Client**: Uses hashicorp/go-retryablehttp with 3 retries by default (configured in main.go:42). This provides automatic retry logic with exponential backoff.

**GraphQL Query**: The usage query is hardcoded as a string constant `usageBody` in main.go.

**MQTT Publishing**: Publishes a single float value (usage in GB formatted to 2 decimal places) to the configured state topic with QoS 1 and retain flag set to true. The connection uses autopaho for automatic reconnection handling.

## Configuration

Required environment variables or flags:
- `CLIENT_SECRET`: OAuth client secret
- `REFRESH_TOKEN`: OAuth refresh token
- `MQTT_URL`: MQTT broker URL (e.g. `mqtt://host:1883`)
- `MQTT_USERNAME`: MQTT username
- `MQTT_PASSWORD`: MQTT password

Optional flags with defaults:
- `-timeout`: Request timeout (default: `90s`)
- `-client_id`: OAuth client ID (default: `xfinity-android-application`)
- `-mqtt_client_id`: MQTT client ID (default: `xfinity-usage-go`)
- `-mqtt_state_topic`: MQTT topic (default: `homeassistant/sensor/xfinity_internet/state`)
- `-prometheus_job`: Prometheus job name (default: `xfinity-usage`)
- `-application_id`: OAuth application ID (optional, from env `APPLICATION_ID`)
- `-prometheus_endpoint`: Prometheus Pushgateway endpoint (optional, from env `PROMETHEUS_ENDPOINT`)

## Dependencies

Key external dependencies:
- `github.com/eclipse/paho.golang`: MQTT client library
- `github.com/hashicorp/go-retryablehttp`: HTTP client with retry logic
- `golang.org/x/oauth2`: OAuth2 token structures
- `github.com/prometheus/client_golang`: Prometheus client for metrics collection and pushing

## Metrics and Observability

The application exports Prometheus metrics to a Pushgateway for monitoring CronJob execution. Metrics are pushed at the end of every run (even on failure) to ensure visibility into job health.

**Available metrics**:

*Counters:*
- `xfinity_usage_runs_total`: Total number of runs
- `xfinity_usage_runs_success_total`: Total successful runs
- `xfinity_usage_errors_total{category}`: Errors by category
- `xfinity_usage_retries_total{host,method,status_code}`: HTTP retry attempts by host, method (GET/POST), and status code (500, 503, 0 for network errors, etc.)

*Gauges:*
- `xfinity_usage_last_success_timestamp`: Unix timestamp of last successful run
- `xfinity_usage_last_run_timestamp`: Unix timestamp of last run (success or failure)
- `xfinity_usage_consecutive_failures`: Number of consecutive failures since last success
- `xfinity_usage_last_run_success`: Whether last run succeeded (1) or failed (0)
- `xfinity_usage_build_info{version,go_version}`: Build information (always 1)

*Histograms:*
- `xfinity_usage_execution_duration_seconds`: Total execution duration
- `xfinity_usage_token_refresh_duration_seconds`: OAuth token refresh duration
- `xfinity_usage_usage_fetch_duration_seconds`: GraphQL usage fetch duration
- `xfinity_usage_mqtt_publish_duration_seconds`: MQTT publish duration

Note: Usage data (current GB, allowable GB, days remaining) is published to MQTT and not duplicated in Prometheus metrics. Prometheus metrics focus on job health and execution monitoring.

**Error categories** tracked via `xfinity_usage_errors_total{category="..."}`:
- `config_validation`: Configuration validation failures
- `token_refresh`: OAuth token refresh failures
- `usage_fetch`: Failures fetching data from GraphQL API
- `usage_parse`: Failures parsing usage response
- `mqtt_publish`: MQTT publishing failures
- `metrics_push`: Prometheus push failures

**Monitoring strategy**:

*Critical alerts:*
- **Job stopped running**: `time() - xfinity_usage_last_run_timestamp > 7200` (no runs in 2 hours)
- **Consecutive failures**: `xfinity_usage_consecutive_failures >= 3` (3+ failures in a row)
- **No successful runs**: `rate(xfinity_usage_runs_success_total[2h]) == 0` (no success in 2 hours)
- **Last run failed**: `xfinity_usage_last_run_success == 0` (simple boolean check)

*Performance alerts:*
- **Token refresh slow**: `histogram_quantile(0.99, rate(xfinity_usage_token_refresh_duration_seconds_bucket[5m])) > 30` (p99 >30s)
- **Usage fetch slow**: `histogram_quantile(0.99, rate(xfinity_usage_usage_fetch_duration_seconds_bucket[5m])) > 30` (p99 >30s)
- **High retry rate**: `rate(xfinity_usage_retries_total[5m]) > 0.5` (frequent retries indicate API issues)

*Diagnostic metrics:*
- Error category counters help identify which component is failing
- Duration histograms help identify performance bottlenecks
- Build info tracks which version is deployed

**Pushgateway setup**: Metrics are pushed using the Prometheus Pushgateway pattern, which is appropriate for short-lived jobs like CronJobs. Configure `PROMETHEUS_ENDPOINT` to point to your Pushgateway instance (e.g., `http://pushgateway:9091`).

## Deployment

The primary deployment target is Kubernetes CronJob (see README.md for full YAML). The container runs as non-root (UID 1000) with read-only root filesystem and all capabilities dropped for security.