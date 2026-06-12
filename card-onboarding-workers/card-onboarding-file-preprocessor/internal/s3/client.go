package s3

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client interface defines methods for downloading/uploading files to S3.
type S3Client interface {
	DownloadFile(ctx context.Context, bucket, key string) ([]byte, error)
	UploadFile(ctx context.Context, bucket, key string, data []byte) error
}

// AwsS3Client wraps AWS SDK v3 S3 client calls.
type AwsS3Client struct {
	client *s3.Client
}

// NewS3Client creates a new AwsS3Client.
func NewS3Client(client *s3.Client) *AwsS3Client {
	return &AwsS3Client{
		client: client,
	}
}

// DownloadFile downloads file content from S3.
func (c *AwsS3Client) DownloadFile(ctx context.Context, bucket, key string) ([]byte, error) {
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	return io.ReadAll(out.Body)
}

// UploadFile uploads file content to S3.
func (c *AwsS3Client) UploadFile(ctx context.Context, bucket, key string, data []byte) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}
