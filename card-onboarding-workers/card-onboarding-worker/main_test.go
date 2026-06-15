package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/client"
	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/config"
	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/validator"
)

type MockOnboardClient struct {
	OnboardFunc func(ctx context.Context, rec validator.CardRecord) error
	CallCount   int
}

func (m *MockOnboardClient) OnboardCard(ctx context.Context, rec validator.CardRecord) error {
	m.CallCount++
	if m.OnboardFunc != nil {
		return m.OnboardFunc(ctx, rec)
	}
	return nil
}

func TestHandleEvent_ScenarioSuccess(t *testing.T) {
	appConfig := &config.Config{
		AwsRegion:         "ap-southeast-2",
		OnboardServiceUrl: "http://localhost:8080",
		TimeoutSeconds:    5,
	}

	mockClient := &MockOnboardClient{
		OnboardFunc: func(ctx context.Context, rec validator.CardRecord) error {
			return nil
		},
	}

	handler := NewWorkerHandler(appConfig, mockClient)

	rec := validator.CardRecord{
		CorrelationId: "corr-1",
		CustomerId:    "CUST001",
		CardType:      "VISA",
		CardNumber:    "4111111111111111",
		ExpiryDate:    "12/28",
		Email:         "test@domain.co",
	}
	bodyBytes, _ := json.Marshal(rec)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				Body: string(bodyBytes),
			},
		},
	}

	err := handler.HandleEvent(context.Background(), sqsEvent)
	if err != nil {
		t.Fatalf("Expected nil error for happy path, got: %v", err)
	}

	if mockClient.CallCount != 1 {
		t.Errorf("Expected client to be called 1 time, got: %d", mockClient.CallCount)
	}
}

func TestHandleEvent_ScenarioValidationFailure(t *testing.T) {
	appConfig := &config.Config{}
	mockClient := &MockOnboardClient{}
	handler := NewWorkerHandler(appConfig, mockClient)

	rec := validator.CardRecord{
		CorrelationId: "corr-2",
		CustomerId:    "CUST002",
		CardType:      "VISA",
		CardNumber:    "4111111111111111",
		ExpiryDate:    "12/24", // Expired
		Email:         "test@domain.co",
	}
	bodyBytes, _ := json.Marshal(rec)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				Body: string(bodyBytes),
			},
		},
	}

	err := handler.HandleEvent(context.Background(), sqsEvent)
	if err != nil {
		t.Fatalf("Expected nil error for business validation failure (so SQS deletes the msg), got: %v", err)
	}

	if mockClient.CallCount != 0 {
		t.Errorf("Expected client NOT to be called, but got call count: %d", mockClient.CallCount)
	}
}

func TestHandleEvent_ScenarioNonRetryableError(t *testing.T) {
	appConfig := &config.Config{}
	mockClient := &MockOnboardClient{
		OnboardFunc: func(ctx context.Context, rec validator.CardRecord) error {
			return &client.OnboardError{
				StatusCode: 400,
				Code:       "VALIDATION_ERROR",
				Message:    "Invalid client parameters",
			}
		},
	}
	handler := NewWorkerHandler(appConfig, mockClient)

	rec := validator.CardRecord{
		CustomerId: "CUST003",
		CardType:   "VISA",
		CardNumber: "4111111111111111",
		ExpiryDate: "12/28",
		Email:      "test@domain.co",
	}
	bodyBytes, _ := json.Marshal(rec)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				Body: string(bodyBytes),
			},
		},
	}

	err := handler.HandleEvent(context.Background(), sqsEvent)
	if err != nil {
		t.Fatalf("Expected nil error for 400 Non-retryable error, got: %v", err)
	}
}

func TestHandleEvent_ScenarioRetryableError(t *testing.T) {
	appConfig := &config.Config{}
	mockClient := &MockOnboardClient{
		OnboardFunc: func(ctx context.Context, rec validator.CardRecord) error {
			return &client.OnboardError{
				StatusCode: 503,
				Code:       "SERVICE_UNAVAILABLE",
				Message:    "Database connection failure",
			}
		},
	}
	handler := NewWorkerHandler(appConfig, mockClient)

	rec := validator.CardRecord{
		CustomerId: "CUST004",
		CardType:   "VISA",
		CardNumber: "4111111111111111",
		ExpiryDate: "12/28",
		Email:      "test@domain.co",
	}
	bodyBytes, _ := json.Marshal(rec)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				Body: string(bodyBytes),
			},
		},
	}

	err := handler.HandleEvent(context.Background(), sqsEvent)
	if err == nil {
		t.Fatal("Expected non-nil error to trigger SQS retries for 503 status code, got nil")
	}
}

func TestHandleEvent_ScenarioNetworkError(t *testing.T) {
	appConfig := &config.Config{}
	mockClient := &MockOnboardClient{
		OnboardFunc: func(ctx context.Context, rec validator.CardRecord) error {
			return errors.New("connection reset by peer")
		},
	}
	handler := NewWorkerHandler(appConfig, mockClient)

	rec := validator.CardRecord{
		CustomerId: "CUST005",
		CardType:   "VISA",
		CardNumber: "4111111111111111",
		ExpiryDate: "12/28",
		Email:      "test@domain.co",
	}
	bodyBytes, _ := json.Marshal(rec)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				Body: string(bodyBytes),
			},
		},
	}

	err := handler.HandleEvent(context.Background(), sqsEvent)
	if err == nil {
		t.Fatal("Expected non-nil error to trigger SQS retries for network connection failure, got nil")
	}
}
