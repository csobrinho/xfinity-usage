package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/google/logger"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	usageExtraHeaders = map[string]string{
		"x-apollo-operation-name": "InternetDataUsage",
		"x-apollo-operation-id":   "61994c6016ac8c0ebcca875084919e5e01cb3b116a86aaf9646e597c3a1fbd06",
		"accept":                  "multipart/mixed; deferSpec=20220824, application/json",
		"user-agent":              "Digital Home / Samsung SM-G991B / Android 14",
		"client":                  "digital-home-android",
		"client-detail":           "MOBILE;Samsung;SM-G991B;Android 14;v5.38.0",
		"accept-language":         "en-US",
		"content-type":            "application/json",
	}
)

type Usage struct {
	Data *struct {
		Account *struct {
			Internet *struct {
				Usage *struct {
					InPaidOverage *bool `json:"inPaidOverage,omitempty"`
					Courtesy      *struct {
						TotalAllowableCourtesy *int `json:"totalAllowableCourtesy,omitempty"`
						UsedCourtesy           *int `json:"usedCourtesy,omitempty"`
						RemainingCourtesy      *int `json:"remainingCourtesy,omitempty"`
					} `json:"courtesy,omitempty"`
					MonthlyUsage []UsageMonthly `json:"monthlyUsage,omitempty"`
				} `json:"usage,omitempty"`
			} `json:"internet,omitempty"`
		} `json:"accountByServiceAccountId,omitempty"`
	} `json:"data,omitempty"`
}

type UsageMonthly struct {
	Policy               string     `json:"policy,omitempty"`
	Month                *int       `json:"month,omitempty"`
	Year                 *int       `json:"year,omitempty"`
	StartDate            string     `json:"startDate,omitempty"`
	EndDate              string     `json:"endDate,omitempty"`
	DaysRemaining        *int       `json:"daysRemaining,omitempty"`
	CurrentUsage         UsageValue `json:"currentUsage,omitempty"`
	AllowableUsage       UsageValue `json:"allowableUsage,omitempty"`
	Overage              bool       `json:"overage"`
	OverageCharge        *int       `json:"overageCharge,omitempty"`
	MaximumOverageCharge *int       `json:"maximumOverageCharge,omitempty"`
	CourtesyCredit       bool       `json:"courtesyCredit"`
}

type UsageValue struct {
	Value *float32 `json:"value,omitempty"`
	Unit  string   `json:"unit,omitempty"`
}

func (u UsageValue) GB() (float32, error) {
	if u.Value == nil {
		return 0, fmt.Errorf("no usage value")
	}
	switch strings.ToLower(u.Unit) {
	case "mb":
		return *u.Value / 1000, nil
	case "gb":
		return *u.Value, nil
	case "tb":
		return *u.Value * 1000, nil
	default:
		return 0, fmt.Errorf("unknown %s unit", u.Unit)
	}
}

// calculateEstimatedUsage calculates the estimated usage at the end of the billing period
// based on current consumption rate. Returns (estimatedUsage, dailyAverage).
func calculateEstimatedUsage(currentGB float32, startDate, endDate string) (float32, float32) {
	// Default to current usage if we can't calculate.
	if startDate == "" || endDate == "" {
		log.Warning("usage: start_date or end_date is empty, cannot calculate estimated usage")
		return currentGB, 0
	}

	start, errStart := time.Parse("2006-01-02", startDate)
	end, errEnd := time.Parse("2006-01-02", endDate)

	if errStart != nil || errEnd != nil {
		log.Warningf("usage: failed to parse dates (start: %q, end: %q): %v, %v", startDate, endDate, errStart, errEnd)
		return currentGB, 0
	}

	now := time.Now()
	totalDays := end.Sub(start).Hours() / 24
	daysElapsed := now.Sub(start).Hours() / 24

	// Ensure we don't divide by zero and days elapsed is positive.
	if daysElapsed <= 0 || totalDays <= 0 {
		log.Warningf("usage: invalid days calculation (elapsed: %.2f, total: %.2f)", daysElapsed, totalDays)
		return currentGB, 0
	}

	dailyAverage := currentGB / float32(daysElapsed)
	estimatedUsage := dailyAverage * float32(totalDays)
	return estimatedUsage, dailyAverage
}

// ToAttributes converts the Usage data to UsageAttributes for MQTT publishing.
func (u *Usage) ToAttributes() (*UsageAttributes, error) {
	// Validate usage data structure.
	if u.Data == nil || u.Data.Account == nil || u.Data.Account.Internet == nil ||
		u.Data.Account.Internet.Usage == nil || len(u.Data.Account.Internet.Usage.MonthlyUsage) == 0 {
		return nil, fmt.Errorf("invalid usage data structure")
	}

	monthlyUsage := u.Data.Account.Internet.Usage.MonthlyUsage[0]

	// Get current and allowable usage in GB.
	currentGB, err := monthlyUsage.CurrentUsage.GB()
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage in GB: %w", err)
	}

	allowableGB, err := monthlyUsage.AllowableUsage.GB()
	if err != nil {
		return nil, fmt.Errorf("failed to get allowable usage in GB: %w", err)
	}
	usageRemaining := max(int(allowableGB-currentGB), 0)

	intPtrToInt := func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	}

	// Extract optional fields with defaults.
	overageCharge := intPtrToInt(monthlyUsage.OverageCharge)
	maxOverageCharge := intPtrToInt(monthlyUsage.MaximumOverageCharge)

	var inPaidOverage bool
	if u.Data.Account.Internet.Usage.InPaidOverage != nil {
		inPaidOverage = *u.Data.Account.Internet.Usage.InPaidOverage
	}

	// Calculate how much over the limit in GB.
	overageUsed := 0
	if monthlyUsage.Overage {
		overageUsed = max(int(currentGB-allowableGB), 0)
	}

	// Calculate estimated usage and daily average.
	usageEstimated, usageDailyAverage := calculateEstimatedUsage(currentGB, monthlyUsage.StartDate, monthlyUsage.EndDate)

	return &UsageAttributes{
		FriendlyName:      "Xfinity Usage",
		UnitOfMeasurement: "GB",
		DeviceClass:       "data_size",
		StateClass:        "measurement",
		Icon:              "mdi:wan",
		//
		StartDate:            monthlyUsage.StartDate,
		EndDate:              monthlyUsage.EndDate,
		DaysRemaining:        intPtrToInt(monthlyUsage.DaysRemaining),
		UsageRemaining:       usageRemaining,
		UsageEstimated:       usageEstimated,
		UsageDailyAverage:    usageDailyAverage,
		AllowableUsage:       int(allowableGB),
		InPaidOverage:        inPaidOverage,
		OverageCharges:       overageCharge,
		OverageUsed:          overageUsed,
		MaximumOverageCharge: maxOverageCharge,
		Policy:               monthlyUsage.Policy,
	}, nil
}

// UsageAttributes represents the usage data published to MQTT for Home Assistant.
type UsageAttributes struct {
	// Main Home Assistant attributes.
	FriendlyName      string `json:"friendly_name"`
	UnitOfMeasurement string `json:"unit_of_measurement"`
	DeviceClass       string `json:"device_class"`
	StateClass        string `json:"state_class"`
	Icon              string `json:"icon"`

	// Custom attributes below.
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	DaysRemaining        int     `json:"days_remaining"`
	UsageRemaining       int     `json:"usage_remaining"`
	UsageEstimated       float32 `json:"usage_estimated"`
	UsageDailyAverage    float32 `json:"usage_daily_average"`
	AllowableUsage       int     `json:"allowable_usage"`
	InPaidOverage        bool    `json:"in_paid_overage"`
	OverageCharges       int     `json:"overage_charges"`
	OverageUsed          int     `json:"overage_used"`
	MaximumOverageCharge int     `json:"maximum_overage_charge"`
	Policy               string  `json:"policy"`
}

func query(ctx context.Context, client *retryablehttp.Client, accessToken, url, method string, requestBody io.Reader, headers map[string]string) ([]byte, error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, method, url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("authorization", "Bearer "+accessToken)
	req.Header.Set("x-id-token", accessToken)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	// Check for HTTP errors
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed with request status %d: %s", res.StatusCode, body)
	}
	return body, nil
}

func internetDataUsageRequest(ctx context.Context, client *retryablehttp.Client, accessToken string) (*Usage, error) {
	body, err := query(ctx, client, accessToken, usageURL, "POST", strings.NewReader(usageBody), usageExtraHeaders)
	if err != nil {
		return nil, err
	}
	// Parse the token response
	u := new(Usage)
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(u); err != nil {
		return nil, fmt.Errorf("failed to parse usage response: %w", err)
	}
	return u, nil
}
