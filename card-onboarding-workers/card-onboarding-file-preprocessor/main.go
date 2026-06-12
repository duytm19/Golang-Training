package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/org/card-onboarding-workers/card-onboarding-file-preprocessor/internal/config"
	"github.com/org/card-onboarding-workers/card-onboarding-file-preprocessor/internal/parser"
	"github.com/org/card-onboarding-workers/card-onboarding-file-preprocessor/internal/s3"
	"github.com/org/card-onboarding-workers/card-onboarding-file-preprocessor/internal/sqs"
)

type PreprocessorHandler struct {
	appConfig *config.Config
	s3Client  s3.S3Client
	sqsClient sqs.SqsClient
}

func NewPreprocessorHandler(appConfig *config.Config, s3Client s3.S3Client, sqsClient sqs.SqsClient) *PreprocessorHandler {
	return &PreprocessorHandler{
		appConfig: appConfig,
		s3Client:  s3Client,
		sqsClient: sqsClient,
	}
}

// HandleEvent processes incoming SQS events that contain S3 upload notifications.
func (h *PreprocessorHandler) HandleEvent(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, sqsRecord := range sqsEvent.Records {
		// Parse S3 Event Notification from SQS body
		var s3Event events.S3Event
		err := json.Unmarshal([]byte(sqsRecord.Body), &s3Event)
		if err != nil {
			log.Printf(`{"level":"error","message":"Failed to unmarshal SQS record body as S3Event: %v"}`+"\n", err)
			return fmt.Errorf("failed to parse S3Event: %w", err)
		}

		for _, s3Record := range s3Event.Records {
			bucketName := s3Record.S3.Bucket.Name
			objectKey := s3Record.S3.Object.Key

			// Decode URL-encoded object key
			decodedKey, err := url.QueryUnescape(objectKey)
			if err == nil {
				objectKey = decodedKey
			}

			log.Printf(`{"level":"info","message":"Downloading file","bucket":"%s","key":"%s"}`+"\n", bucketName, objectKey)

			// Download CSV data from S3
			data, err := h.s3Client.DownloadFile(ctx, bucketName, objectKey)
			if err != nil {
				log.Printf(`{"level":"error","message":"Failed to download file from S3: %v"}`+"\n", err)
				return fmt.Errorf("failed to download file from S3: %w", err)
			}

			// Generate a unique Job ID
			jobId := fmt.Sprintf("JOB-%d", time.Now().UnixNano())
			fileName := filepath.Base(objectKey)

			// Validate and parse the CSV
			parsedRecords, resultCSV, err := parser.ValidateCSV(data, fileName, jobId, h.appConfig.MaxFileSizeBytes)
			if err != nil {
				// Fatal structural validation error (extension, size, header mismatch, corrupt parse)
				log.Printf(`{"level":"error","message":"File structural validation failed: %v"}`+"\n", err)
				return fmt.Errorf("structural validation failed for file %s: %w", fileName, err)
			}

			// Publish accepted records to worker SQS queue
			for _, rec := range parsedRecords {
				corrID := "corr-" + rec.RecordId
				payload := sqs.WorkerPayload{
					CorrelationId: corrID,
					JobId:         jobId,
					RecordId:      rec.RecordId,
					SourceFile:    fileName,
					RowNumber:     rec.RowNumber,
					CustomerId:    rec.CustomerId,
					CardType:      rec.CardType,
					CardNumber:    rec.CardNumber,
					ExpiryDate:    rec.ExpiryDate,
					HolderName:    rec.HolderName,
					Email:         rec.Email,
				}

				err = h.sqsClient.PublishMessage(ctx, h.appConfig.WorkerSqsUrl, payload)
				if err != nil {
					log.Printf(`{"level":"error","message":"Failed to publish record to SQS: %v"}`+"\n", err)
					return fmt.Errorf("failed to publish record %s: %w", rec.RecordId, err)
				}
			}

			// Upload preprocessing result summary CSV to S3 output bucket
			resultKey := fmt.Sprintf("processed/%s/%s_preprocess_result.csv", jobId, fileName)
			log.Printf(`{"level":"info","message":"Uploading result summary","bucket":"%s","key":"%s"}`+"\n", h.appConfig.OutputBucket, resultKey)

			err = h.s3Client.UploadFile(ctx, h.appConfig.OutputBucket, resultKey, resultCSV)
			if err != nil {
				log.Printf(`{"level":"error","message":"Failed to upload result to S3: %v"}`+"\n", err)
				return fmt.Errorf("failed to upload validation results: %w", err)
			}
		}
	}
	return nil
}

func main() {
	appConfig := config.Load()

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(appConfig.AwsRegion))
	if err != nil {
		log.Printf(`{"level":"warn","message":"Failed to load default AWS config: %v"}`+"\n", err)
	}

	s3Client := s3.NewS3Client(awss3.NewFromConfig(awsCfg))
	sqsClient := sqs.NewSqsClient(awssqs.NewFromConfig(awsCfg))

	handler := NewPreprocessorHandler(appConfig, s3Client, sqsClient)
	lambda.Start(handler.HandleEvent)
}
