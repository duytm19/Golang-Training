package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/client"
	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/config"
	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/logutil"
	"github.com/org/card-onboarding-workers/card-onboarding-worker/internal/validator"
)

type WorkerHandler struct {
	appConfig     *config.Config
	onboardClient client.OnboardClient
}

func NewWorkerHandler(appConfig *config.Config, onboardClient client.OnboardClient) *WorkerHandler {
	return &WorkerHandler{
		appConfig:     appConfig,
		onboardClient: onboardClient,
	}
}

func (h *WorkerHandler) HandleEvent(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		var rec validator.CardRecord
		err := json.Unmarshal([]byte(record.Body), &rec)
		if err != nil {
			log.Printf(`{"level":"error","message":"Failed to unmarshal SQS record body: %v"}`+"\n", err)
			continue
		}

		maskedCard := logutil.MaskCardNumber(rec.CardNumber)
		log.Printf(`{"level":"info","message":"Processing card record","correlationId":"%s","jobId":"%s","recordId":"%s","customerId":"%s","cardNumber":"%s"}`+"\n",
			rec.CorrelationId, rec.JobId, rec.RecordId, rec.CustomerId, maskedCard)

		// 1. Business validations
		err = validator.ValidateCardRecord(rec, time.Now())
		if err != nil {
			log.Printf(`{"level":"warn","message":"Business validation failed for card: %v","correlationId":"%s","recordId":"%s","customerId":"%s"}`+"\n",
				err, rec.CorrelationId, rec.RecordId, rec.CustomerId)
			continue
		}

		// 2. Call onboard service orchestrator
		err = h.onboardClient.OnboardCard(ctx, rec)
		if err != nil {
			var onboardErr *client.OnboardError
			if errors.As(err, &onboardErr) {
				if onboardErr.IsRetryable() {
					log.Printf(`{"level":"error","message":"Retryable onboard service error: %v","correlationId":"%s","recordId":"%s"}`+"\n",
						onboardErr, rec.CorrelationId, rec.RecordId)
					return fmt.Errorf("retryable onboard error: %w", err)
				} else {
					log.Printf(`{"level":"warn","message":"Non-retryable onboard service error: %v","correlationId":"%s","recordId":"%s"}`+"\n",
						onboardErr, rec.CorrelationId, rec.RecordId)
					continue
				}
			}

			log.Printf(`{"level":"error","message":"Network or timeout error calling onboard service: %v","correlationId":"%s","recordId":"%s"}`+"\n",
				err, rec.CorrelationId, rec.RecordId)
			return fmt.Errorf("retryable network error: %w", err)
		}

		log.Printf(`{"level":"info","message":"Card record onboarded successfully","correlationId":"%s","recordId":"%s","customerId":"%s"}`+"\n",
			rec.CorrelationId, rec.RecordId, rec.CustomerId)
	}

	return nil
}

func main() {
	appConfig := config.Load()
	onboardClient := client.NewOnboardClient(appConfig.OnboardServiceUrl, time.Duration(appConfig.TimeoutSeconds)*time.Second)

	handler := NewWorkerHandler(appConfig, onboardClient)
	lambda.Start(handler.HandleEvent)
}
