package config

import (
	"os"
)

// Config holds all configuration parameters for onboard-service
type Config struct {
	Port                string
	CustomerServiceURL  string
	AccountServiceURL   string
	AwsRegion           string
	RequestStatusTable  string
	AccountDetailsTable string
	DatadogAgentAddr    string
}

// Load loads configurations from environment variables or returns defaults
func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		CustomerServiceURL:  getEnv("CUSTOMER_SERVICE_URL", "http://localhost:8081"),
		AccountServiceURL:   getEnv("ACCOUNT_SERVICE_URL", "http://localhost:8082"),
		AwsRegion:           getEnv("AWS_REGION", "ap-southeast-2"),
		RequestStatusTable:  getEnv("REQUEST_STATUS_TABLE", "onboard-service-request-status"),
		AccountDetailsTable: getEnv("ACCOUNT_DETAILS_TABLE", "onboard-service-account-details"),
		DatadogAgentAddr:    getEnv("DATADOG_AGENT_ADDR", "localhost:8125"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
