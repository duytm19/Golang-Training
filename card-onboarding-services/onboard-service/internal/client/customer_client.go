package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/org/card-onboarding-services/customer-management-service/pkg/customer"
)

// CustomerClient defines methods for calling the customer registry.
type CustomerClient interface {
	RegisterCustomer(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customer.RegisterCustomerResult, error)
	GetCustomer(ctx context.Context, correlationID string, customerID string) (*customer.RegisterCustomerResult, error)
}

// HttpCustomerClient wraps customer service HTTP calls.
type HttpCustomerClient struct {
	baseURL string
}

// NewHttpCustomerClient creates a new HttpCustomerClient.
func NewHttpCustomerClient(baseURL string) *HttpCustomerClient {
	return &HttpCustomerClient{
		baseURL: baseURL,
	}
}

// RegisterCustomer calls POST /internal/customers/register.
func (c *HttpCustomerClient) RegisterCustomer(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customer.RegisterCustomerResult, error) {
	cli, err := customer.NewClientWithResponses(c.baseURL, customer.WithHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	if err != nil {
		return nil, err
	}

	reqBody := customer.RegisterCustomerRequest{
		CorrelationId: correlationID,
		CustomerId:    customerID,
		HolderName:    holderName,
		Email:         openapi_types.Email(email),
	}

	res, err := cli.RegisterCustomerWithResponse(ctx, reqBody, func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Correlation-Id", correlationID)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		if res.JSON500 != nil {
			return nil, fmt.Errorf("customer registry register error 500: %s (code: %s)", res.JSON500.Message, res.JSON500.Code)
		}
		if res.JSON400 != nil {
			return nil, fmt.Errorf("customer registry register error 400: %s (code: %s)", res.JSON400.Message, res.JSON400.Code)
		}
		return nil, fmt.Errorf("customer registry register error: status code %d", res.StatusCode())
	}

	return res.JSON200, nil
}

// GetCustomer calls GET /internal/customers/{customerId}.
func (c *HttpCustomerClient) GetCustomer(ctx context.Context, correlationID string, customerID string) (*customer.RegisterCustomerResult, error) {
	cli, err := customer.NewClientWithResponses(c.baseURL, customer.WithHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	if err != nil {
		return nil, err
	}

	res, err := cli.GetCustomerWithResponse(ctx, customerID, func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Correlation-Id", correlationID)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		if res.JSON500 != nil {
			return nil, fmt.Errorf("customer registry get error 500: %s (code: %s)", res.JSON500.Message, res.JSON500.Code)
		}
		if res.JSON404 != nil {
			return nil, fmt.Errorf("customer registry get error 404: %s (code: %s)", res.JSON404.Message, res.JSON404.Code)
		}
		return nil, fmt.Errorf("customer registry get error: status code %d", res.StatusCode())
	}

	return res.JSON200, nil
}
