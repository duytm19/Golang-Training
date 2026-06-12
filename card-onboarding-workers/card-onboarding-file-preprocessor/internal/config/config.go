package config

import (
	"os"
	"strconv"
)

// Config stores configurations for the file preprocessor worker.
type Config struct {
	AwsRegion        string
	OutputBucket     string
	WorkerSqsUrl     string
	MaxFileSizeBytes int64
}

// Load reads config variables from the environment.
func Load() *Config {
	maxSize := getEnvAsInt64("MAX_FILE_SIZE_BYTES", 10485760) // Default 10MB
	return &Config{
		AwsRegion:        getEnv("AWS_REGION", "ap-southeast-2"),
		OutputBucket:     getEnv("OUTPUT_BUCKET", "card-onboarding-output-bucket-dev"),
		WorkerSqsUrl:     getEnv("WORKER_SQS_URL", "https://sqs.ap-southeast-2.amazonaws.com/123456789012/card-onboarding-worker-sqs-dev"),
		MaxFileSizeBytes: maxSize,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvAsInt64(key string, fallback int64) int64 {
	valStr := getEnv(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return fallback
	}
	return val
}
