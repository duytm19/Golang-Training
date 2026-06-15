package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/validator"
)

type OnboardClient interface {
	OnboardCard(ctx context.Context, rec validator.CardRecord) error
}

type HttpOnboardClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewOnboardClient(baseURL string, timeout time.Duration) *HttpOnboardClient {
	return &HttpOnboardClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HttpOnboardClient) OnboardCard(ctx context.Context, rec validator.CardRecord) error {
	url := fmt.Sprintf("%s/internal/cards/onboard", c.baseURL)

	payloadBytes, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if rec.CorrelationId != "" {
		req.Header.Set("X-Correlation-Id", rec.CorrelationId)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	var errResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errResp)

	return &OnboardError{
		StatusCode: resp.StatusCode,
		Code:       errResp.Code,
		Message:    errResp.Message,
	}
}

type OnboardError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *OnboardError) Error() string {
	return fmt.Sprintf("onboard service returned status %d (code=%q, message=%q)", e.StatusCode, e.Code, e.Message)
}

func (e *OnboardError) IsRetryable() bool {
	return e.StatusCode >= 500
}
