package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/org/card-onboarding-services/account-management-service/pkg/account"
)

// AccountClient defines methods for calling the account savings product registry.
type AccountClient interface {
	GetInterestDetails(ctx context.Context, correlationID string, customerID string) (*account.InterestDetailsResponse, error)
}

// HttpAccountClient wraps account service HTTP calls.
type HttpAccountClient struct {
	baseURL string
}

// NewHttpAccountClient creates a new HttpAccountClient.
func NewHttpAccountClient(baseURL string) *HttpAccountClient {
	return &HttpAccountClient{
		baseURL: baseURL,
	}
}

// GetInterestDetails calls GET /internal/accounts/{customerId}/interest-details.
func (c *HttpAccountClient) GetInterestDetails(ctx context.Context, correlationID string, customerID string) (*account.InterestDetailsResponse, error) {
	cli, err := account.NewClientWithResponses(c.baseURL, account.WithHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	if err != nil {
		return nil, err
	}

	res, err := cli.GetInterestDetailsWithResponse(ctx, customerID, func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Correlation-Id", correlationID)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		if res.JSON500 != nil {
			return nil, fmt.Errorf("account registry get error 500: %s (code: %s)", res.JSON500.Message, res.JSON500.Code)
		}
		if res.JSON404 != nil {
			return nil, fmt.Errorf("account registry get error 404: %s (code: %s)", res.JSON404.Message, res.JSON404.Code)
		}
		return nil, fmt.Errorf("account registry get error: status code %d", res.StatusCode())
	}

	return res.JSON200, nil
}
