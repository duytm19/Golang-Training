package config

import (
	"os"
	"strconv"
)

// Config holds the application configurations for the card onboarding processor worker.
type Config struct {
	AwsRegion         string
	OnboardServiceUrl string
	TimeoutSeconds    int
}

// Load reads and parses configuration values from environment variables.
func Load() *Config {
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "ap-southeast-2"
	}

	onboardServiceUrl := os.Getenv("ONBOARD_SERVICE_URL")
	if onboardServiceUrl == "" {
		onboardServiceUrl = "http://localhost:8080"
	}

	timeoutStr := os.Getenv("TIMEOUT_SECONDS")
	timeout := 10
	if timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	return &Config{
		AwsRegion:         awsRegion,
		OnboardServiceUrl: onboardServiceUrl,
		TimeoutSeconds:    timeout,
	}
}
