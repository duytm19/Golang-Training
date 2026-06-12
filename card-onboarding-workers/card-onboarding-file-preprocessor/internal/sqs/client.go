package sqs

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SqsClient defines the interface for publishing events to SQS.
type SqsClient interface {
	PublishMessage(ctx context.Context, queueUrl string, message interface{}) error
}

// AwsSqsClient wraps AWS SDK SQS actions.
type AwsSqsClient struct {
	client *sqs.Client
}

// NewSqsClient creates a new AwsSqsClient.
func NewSqsClient(client *sqs.Client) *AwsSqsClient {
	return &AwsSqsClient{
		client: client,
	}
}

// PublishMessage marshals any payload to JSON and sends it to the target SQS queue.
func (c *AwsSqsClient) PublishMessage(ctx context.Context, queueUrl string, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = c.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueUrl),
		MessageBody: aws.String(string(body)),
	})
	return err
}

// WorkerPayload represents the JSON payload dispatched for each accepted row.
type WorkerPayload struct {
	CorrelationId string `json:"correlationId"`
	JobId         string `json:"jobId"`
	RecordId      string `json:"recordId"`
	SourceFile    string `json:"sourceFile"`
	RowNumber     int    `json:"rowNumber"`
	CustomerId    string `json:"customerId"`
	CardType      string `json:"cardType"`
	CardNumber    string `json:"cardNumber"`
	ExpiryDate    string `json:"expiryDate"`
	HolderName    string `json:"holderName"`
	Email         string `json:"email"`
}
