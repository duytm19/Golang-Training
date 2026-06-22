package appmetrics

import (
	"testing"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

type MockStatsdClient struct {
	statsd.ClientInterface
	IncrCalls         map[string]int
	IncrTags          map[string][]string
	DistributionCalls map[string]int
	DistributionVals  map[string][]float64
	DistributionTags  map[string][]string
}

func NewMockStatsdClient() *MockStatsdClient {
	return &MockStatsdClient{
		IncrCalls:         make(map[string]int),
		IncrTags:          make(map[string][]string),
		DistributionCalls: make(map[string]int),
		DistributionVals:  make(map[string][]float64),
		DistributionTags:  make(map[string][]string),
	}
}

func (m *MockStatsdClient) Incr(name string, tags []string, rate float64) error {
	m.IncrCalls[name]++
	m.IncrTags[name] = tags
	return nil
}

func (m *MockStatsdClient) Distribution(name string, value float64, tags []string, rate float64) error {
	m.DistributionCalls[name]++
	m.DistributionVals[name] = append(m.DistributionVals[name], value)
	m.DistributionTags[name] = tags
	return nil
}

func (m *MockStatsdClient) Close() error {
	return nil
}

func TestEmitMetrics_NilClient(t *testing.T) {
	// Set client to nil
	SetClient(nil)

	// Ensure calling functions does not panic
	EmitRecordAccepted([]string{"tag:val"})
	EmitBusinessValidationFailed([]string{"tag:val"})
	EmitProcessingDuration(100*time.Millisecond, []string{"tag:val"})
}

func TestEmitMetrics_MockClient(t *testing.T) {
	mock := NewMockStatsdClient()
	SetClient(mock)

	EmitRecordAccepted([]string{"accepted_tag"})
	if mock.IncrCalls["records_accepted.count"] != 1 {
		t.Errorf("Expected 1 call to Incr records_accepted.count, got %d", mock.IncrCalls["records_accepted.count"])
	}
	if len(mock.IncrTags["records_accepted.count"]) != 1 || mock.IncrTags["records_accepted.count"][0] != "accepted_tag" {
		t.Errorf("Expected tag accepted_tag, got %v", mock.IncrTags["records_accepted.count"])
	}

	EmitBusinessValidationFailed([]string{"fail_tag"})
	if mock.IncrCalls["business_validation_failed.count"] != 1 {
		t.Errorf("Expected 1 call to Incr business_validation_failed.count, got %d", mock.IncrCalls["business_validation_failed.count"])
	}

	EmitProcessingDuration(150*time.Millisecond, []string{"duration_tag"})
	if mock.DistributionCalls["processing_duration.histogram"] != 1 {
		t.Errorf("Expected 1 call to Distribution processing_duration.histogram, got %d", mock.DistributionCalls["processing_duration.histogram"])
	}
	vals := mock.DistributionVals["processing_duration.histogram"]
	if len(vals) != 1 || vals[0] != 150.0 {
		t.Errorf("Expected recorded value 150.0, got %v", vals)
	}
}
