package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/org/card-onboarding-workers/card-onboarding-file-preprocessor/internal/config"
	"github.com/org/card-onboarding-workers/card-onboarding-file-preprocessor/internal/sqs"
)

type MockS3Client struct {
	DownloadedBucket string
	DownloadedKey    string
	UploadedBucket   string
	UploadedKey      string
	UploadedData     []byte
	CSVData          string
}

func (m *MockS3Client) DownloadFile(ctx context.Context, bucket, key string) ([]byte, error) {
	m.DownloadedBucket = bucket
	m.DownloadedKey = key
	return []byte(m.CSVData), nil
}

func (m *MockS3Client) UploadFile(ctx context.Context, bucket, key string, data []byte) error {
	m.UploadedBucket = bucket
	m.UploadedKey = key
	m.UploadedData = data
	return nil
}

type MockSqsClient struct {
	PublishedMessages []interface{}
}

func (m *MockSqsClient) PublishMessage(ctx context.Context, queueUrl string, message interface{}) error {
	m.PublishedMessages = append(m.PublishedMessages, message)
	return nil
}

func TestHandleEvent_Success(t *testing.T) {
	appConfig := &config.Config{
		AwsRegion:        "ap-southeast-2",
		OutputBucket:     "out-bucket",
		WorkerSqsUrl:     "sqs-url",
		MaxFileSizeBytes: 1000,
	}

	csvData := "customer_id,card_type,card_number,expiry_date,holder_name,email\n" +
		"CUST001,VISA,4111,12/28,Name1,email1@example.com\n" +
		"CUST002,MASTERCARD,5555,10/27,Name2\n" // Malformed row, will be rejected in results

	s3Mock := &MockS3Client{CSVData: csvData}
	sqsMock := &MockSqsClient{}

	handler := NewPreprocessorHandler(appConfig, s3Mock, sqsMock)

	// Construct S3 Event Notification mock
	s3Event := events.S3Event{
		Records: []events.S3EventRecord{
			{
				S3: events.S3Entity{
					Bucket: events.S3Bucket{Name: "in-bucket"},
					Object: events.S3Object{Key: "cards.csv"},
				},
			},
		},
	}
	s3EventBytes, _ := json.Marshal(s3Event)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				Body:      string(s3EventBytes),
				MessageId: "sqs-msg-id",
			},
		},
	}

	err := handler.HandleEvent(context.Background(), sqsEvent)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	// Verify S3 download call parameters
	if s3Mock.DownloadedBucket != "in-bucket" || s3Mock.DownloadedKey != "cards.csv" {
		t.Errorf("Unexpected S3 download: bucket=%s, key=%s", s3Mock.DownloadedBucket, s3Mock.DownloadedKey)
	}

	// Verify S3 upload call parameters
	if s3Mock.UploadedBucket != "out-bucket" || !strings.Contains(s3Mock.UploadedKey, "cards.csv_preprocess_result.csv") {
		t.Errorf("Unexpected S3 upload: bucket=%s, key=%s", s3Mock.UploadedBucket, s3Mock.UploadedKey)
	}

	// Verify SQS messages published (only 1 record CUST001 should be accepted)
	if len(sqsMock.PublishedMessages) != 1 {
		t.Errorf("Expected 1 SQS message published, got %d", len(sqsMock.PublishedMessages))
	} else {
		payload, ok := sqsMock.PublishedMessages[0].(sqs.WorkerPayload)
		if !ok {
			t.Fatalf("Failed to cast message as SqsPayload")
		}
		if payload.CustomerId != "CUST001" {
			t.Errorf("Expected customerId CUST001, got %s", payload.CustomerId)
		}
		if payload.RowNumber != 2 {
			t.Errorf("Expected row number 2, got %d", payload.RowNumber)
		}
		if !strings.HasPrefix(payload.CorrelationId, "corr-REC-") {
			t.Errorf("Expected correlation ID prefixed with corr-REC-, got %s", payload.CorrelationId)
		}
	}
}
