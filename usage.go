package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		return *u.Value / 1024, nil
	case "gb":
		return *u.Value, nil
	case "tb":
		return *u.Value * 1024, nil
	default:
		return 0, fmt.Errorf("unknown %s unit", u.Unit)
	}
}

func internetDataUsageRequest(ctx context.Context, client *http.Client, accessToken string) (*Usage, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", usageURL, strings.NewReader(usageBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("authorization", "Bearer "+accessToken)
	req.Header.Set("x-id-token", accessToken)
	for key, value := range usageExtraHeaders {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usage request failed with status %d: %s", resp.StatusCode, body)
	}

	// Parse the token response
	u := new(Usage)
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(u); err != nil {
		return nil, fmt.Errorf("failed to parse usage response: %w", err)
	}
	return u, nil
}
