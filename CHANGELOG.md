# Changelog

All notable changes to this project are documented in this file.

## [v0.3.5] - 2026-06-22
- Improved attributes parsing logic and fixed a nil pointer dereference when handling unlimited policy attributes.

## [v0.3.4]
- Better support for unlimited policies

**Full Changelog**: https://github.com/csobrinho/xfinity-usage/compare/v0.3.2...v0.3.4

## [v0.3.2]
### Features
- Add `id_token` and `activity_id` propagation from OAuth token response via `TokenExtra`
- Add internet plan data to GraphQL query (`name`, `downloadSpeed`, `uploadSpeed`)

### Fixes
- Fix MQTT client ID using OAuth client ID instead of the configured MQTT client ID
- Fix `clientSecret` being required even when pre-obtained `access_token`/`id_token` are provided
- Fix MQTT disconnect using a potentially cancelled context, causing spurious log noise on shutdown

### Improvements
- Route MQTT debug/error logs to appropriate log levels (`AsDebug`, `AsWarn`)
- Fix `bindArgs` panic on odd-length key-value slices in logger
- Consolidate Prometheus `MustRegister` into a single variadic call
- Use `strconv.Itoa` instead of `fmt.Sprintf` for status code formatting in retry metrics

### Dependencies
- Update `golang.org/x/oauth2` to v0.35.0
- Update `github.com/google/logger` to v1.1.2
- Update Go Docker image to v1.26

**Full Changelog**: https://github.com/csobrinho/xfinity-usage/compare/v0.3.1...v0.3.2

## [v0.3.1]
## Changes
- Add error timestamp gauge for Prometheus alerting using `changes()` function
- Add Prometheus alerting rules in `prometheus/` folder for Kustomize integration
- Optimize Docker build for faster multi-platform images
  - Use native Go cross-compilation instead of QEMU emulation
  - Switch to scratch base image for minimal size
  - Strip debug symbols from binary
  - Add latest tag for main branch builds

**Full Changelog**: https://github.com/csobrinho/xfinity-usage/compare/v0.3.0...v0.3.1

## [v0.3.0]
## What's New

This release adds comprehensive MQTT attributes publishing for Home Assistant, enabling rich usage tracking and projections.

### Features

- **Dual MQTT Publishing**: Separate topics for state (numeric GB value) and attributes (detailed JSON)
- **Usage Projections**: Calculates estimated end-of-month usage based on current consumption rate
- **Daily Average Tracking**: Monitors average GB consumed per day
- **Enhanced Attributes**: 15+ fields including billing dates, overage tracking, and usage metrics

### New MQTT Attributes

The attributes topic now publishes a JSON payload with:
- `start_date`, `end_date` - Billing period dates
- `days_remaining` - Days left in current billing cycle
- `usage_remaining` - GB remaining before hitting allowance
- `usage_estimated` - Projected total usage at end of month
- `usage_daily_average` - Average GB consumed per day
- `allowable_usage` - Monthly data allowance in GB
- `overage_charges` - Current overage charges ($)
- `overage_used` - GB over the allowance limit
- `maximum_overage_charge` - Maximum possible overage charge ($)
- `in_paid_overage` - Whether currently in paid overage
- `policy` - Account policy type
- Home Assistant metadata: `friendly_name`, `unit_of_measurement`, `device_class`, `state_class`, `icon`

### Configuration Changes

**New required parameter:**
- `--mqtt_attributes_topic` - Default: `homeassistant/sensor/xfinity_internet/attributes`

### Bug Fixes

- Fixed GB conversion to use 1000 instead of 1024 for decimal units (matching ISP billing standards)

### Technical Details

The estimated usage calculation uses the billing period dates to compute daily average consumption and project total usage, helping users predict if they'll exceed their allowance before the end of the billing cycle.

**Full Changelog**: https://github.com/csobrinho/xfinity-usage/compare/v0.2.0...v0.3.0

## [v0.2.0]
## What's Changed

### Features
- Add comprehensive Prometheus metrics for CronJob health monitoring
  - Track execution duration, success/failure rates, and consecutive failures
  - Monitor token refresh, usage fetch, and MQTT publish operations
  - HTTP retry tracking by host, method, and status code
  - Build information with version tracking

### Dependencies
- Update actions/setup-go action to v6
- Update dependency go to v1.25.4
