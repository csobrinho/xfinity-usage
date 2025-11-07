package main

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// version is set at build time via -ldflags.
	version = "dev"

	// Counter for total runs.
	runsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "xfinity_usage_runs_total",
		Help: "Total number of xfinity-usage runs",
	})

	// Counter for successful runs.
	runsSuccessTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "xfinity_usage_runs_success_total",
		Help: "Total number of successful xfinity-usage runs",
	})

	// Counter for errors by category.
	errorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "xfinity_usage_errors_total",
		Help: "Total number of errors by category",
	}, []string{"category"})

	// Gauge for last successful run timestamp.
	lastSuccessTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xfinity_usage_last_success_timestamp",
		Help: "Timestamp of the last successful run",
	})

	// Gauge for last run timestamp (success or failure).
	lastRunTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xfinity_usage_last_run_timestamp",
		Help: "Timestamp of the last run (success or failure)",
	})

	// Gauge for consecutive failures.
	consecutiveFailures = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xfinity_usage_consecutive_failures",
		Help: "Number of consecutive failures since last success",
	})

	// Gauge for last run success status.
	lastRunSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xfinity_usage_last_run_success",
		Help: "Whether the last run was successful (1) or failed (0)",
	})

	// Histogram for execution duration.
	executionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "xfinity_usage_execution_duration_seconds",
		Help:    "Execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// Histogram for token refresh duration.
	tokenRefreshDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "xfinity_usage_token_refresh_duration_seconds",
		Help:    "Token refresh operation duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// Histogram for usage fetch duration.
	usageFetchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "xfinity_usage_usage_fetch_duration_seconds",
		Help:    "Usage data fetch operation duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// Histogram for MQTT publish duration.
	mqttPublishDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "xfinity_usage_mqtt_publish_duration_seconds",
		Help:    "MQTT publish operation duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// Counter for retries by host, method, and status code.
	retriesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "xfinity_usage_retries_total",
		Help: "Total number of retries by host, method, and status code",
	}, []string{"host", "method", "status_code"})

	// Gauge for build info.
	buildInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "xfinity_usage_build_info",
		Help: "Build information (version, go_version)",
	}, []string{"version", "go_version"})

	metricsRegistry = prometheus.NewRegistry()
)

func init() {
	// Register all metrics with the custom registry.
	metricsRegistry.MustRegister(runsTotal)
	metricsRegistry.MustRegister(runsSuccessTotal)
	metricsRegistry.MustRegister(errorsTotal)
	metricsRegistry.MustRegister(lastSuccessTimestamp)
	metricsRegistry.MustRegister(lastRunTimestamp)
	metricsRegistry.MustRegister(consecutiveFailures)
	metricsRegistry.MustRegister(lastRunSuccess)
	metricsRegistry.MustRegister(executionDuration)
	metricsRegistry.MustRegister(tokenRefreshDuration)
	metricsRegistry.MustRegister(usageFetchDuration)
	metricsRegistry.MustRegister(mqttPublishDuration)
	metricsRegistry.MustRegister(retriesTotal)
	metricsRegistry.MustRegister(buildInfo)
}

// errorCategory represents an error category for metrics.
type errorCategory string

// Error categories.
const (
	errorCategoryConfigValidation errorCategory = "config_validation"
	errorCategoryTokenRefresh     errorCategory = "token_refresh"
	errorCategoryUsageFetch       errorCategory = "usage_fetch"
	errorCategoryUsageParse       errorCategory = "usage_parse"
	errorCategoryMQTTPublish      errorCategory = "mqtt_publish"
	errorCategoryMetricsPush      errorCategory = "metrics_push"
)

// recordError increments the error counter for a specific category.
func recordError(category errorCategory) {
	errorsTotal.WithLabelValues(string(category)).Inc()
}

// recordSuccess records a successful run.
func recordSuccess() {
	runsSuccessTotal.Inc()
	lastSuccessTimestamp.Set(float64(time.Now().Unix()))
	lastRunSuccess.Set(1)
	consecutiveFailures.Set(0)
}

// recordFailure records a failed run.
func recordFailure() {
	lastRunSuccess.Set(0)
	consecutiveFailures.Inc()
}

// recordRunStart records the start of a run.
func recordRunStart() {
	lastRunTimestamp.Set(float64(time.Now().Unix()))
}

// setBuildInfo sets the build information metric.
func setBuildInfo(version, goVersion string) {
	buildInfo.WithLabelValues(version, goVersion).Set(1)
}

// recordRetry increments the retry counter for a specific host, method, and status code.
func recordRetry(host, method string, statusCode int) {
	retriesTotal.WithLabelValues(host, method, fmt.Sprintf("%d", statusCode)).Inc()
}

// pushMetrics pushes all metrics to the Prometheus Pushgateway.
func pushMetrics(ctx context.Context, endpoint string, job string) error {
	if endpoint == "" {
		// If no endpoint is configured, skip pushing metrics.
		return nil
	}

	pusher := push.New(endpoint, job).Gatherer(metricsRegistry)
	if err := pusher.PushContext(ctx); err != nil {
		return fmt.Errorf("failed to push metrics: %w", err)
	}
	return nil
}
