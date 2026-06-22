package appmetrics

import (
	"log"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

var client statsd.ClientInterface

// Init initializes the Datadog StatsD client.
func Init(addr string) {
	if addr == "" {
		log.Println(`{"level":"info","message":"Datadog StatsD address is empty, metrics collection is disabled"}`)
		return
	}
	c, err := statsd.New(addr, statsd.WithNamespace("card_onboarding."))
	if err != nil {
		log.Printf(`{"level":"warn","message":"Failed to initialize Datadog StatsD client: %v"}`+"\n", err)
		return
	}
	client = c
	log.Printf(`{"level":"info","message":"Datadog StatsD client initialized","addr":"%s"}`+"\n", addr)
}

// Close closes the StatsD client connection.
func Close() {
	if client != nil {
		_ = client.Close()
	}
}

// EmitRecordAccepted increments the records_accepted.count metric.
func EmitRecordAccepted(tags []string) {
	if client != nil {
		_ = client.Incr("records_accepted.count", tags, 1.0)
	}
}

// EmitBusinessValidationFailed increments the business_validation_failed.count metric.
func EmitBusinessValidationFailed(tags []string) {
	if client != nil {
		_ = client.Incr("business_validation_failed.count", tags, 1.0)
	}
}

// EmitProcessingDuration records the processing duration histogram in milliseconds.
func EmitProcessingDuration(duration time.Duration, tags []string) {
	if client != nil {
		_ = client.Distribution("processing_duration.histogram", float64(duration.Milliseconds()), tags, 1.0)
	}
}

// SetClient allows injecting a mock client interface for testing.
func SetClient(c statsd.ClientInterface) {
	client = c
}
