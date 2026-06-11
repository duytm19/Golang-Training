package store

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockRequestStatusStore implements RequestStatusStore in memory.
type MockRequestStatusStore struct {
	mu     sync.RWMutex
	status map[string]*RequestStatus
}

// NewMockRequestStatusStore creates a new MockRequestStatusStore.
func NewMockRequestStatusStore() *MockRequestStatusStore {
	return &MockRequestStatusStore{
		status: make(map[string]*RequestStatus),
	}
}

// GetRequestStatus retrieves status from memory.
func (m *MockRequestStatusStore) GetRequestStatus(ctx context.Context, customerId string) (*RequestStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.status[customerId]
	if !ok {
		return nil, nil
	}
	cp := *val
	return &cp, nil
}

// CreateRequestStatus initializes status in memory.
func (m *MockRequestStatusStore) CreateRequestStatus(ctx context.Context, customerId, jobId, recordId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.status[customerId]; ok {
		return errors.New("ConditionalCheckFailedException: customerId already exists")
	}
	m.status[customerId] = &RequestStatus{
		CustomerId:                 customerId,
		JobId:                      jobId,
		RecordId:                   recordId,
		OverallStatus:              "IN_PROGRESS",
		CustomerRegistrationStatus: "IN_PROGRESS",
		UpdatedAt:                  time.Now().UTC(),
	}
	return nil
}

// UpdateCustomerRegistration updates customer step in memory.
func (m *MockRequestStatusStore) UpdateCustomerRegistration(ctx context.Context, customerId string, status string, coreCustomerId string, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.status[customerId]
	if !ok {
		return errors.New("ConditionalCheckFailedException: customerId does not exist")
	}
	val.CustomerRegistrationStatus = status
	val.CustomerRegistrationMessage = message
	if coreCustomerId != "" {
		val.CoreCustomerId = coreCustomerId
	}
	if status == "FAILED" || status == "IN_PROGRESS" {
		val.OverallStatus = status
	}
	val.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateInterestDetails updates interest step in memory.
func (m *MockRequestStatusStore) UpdateInterestDetails(ctx context.Context, customerId string, status string, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.status[customerId]
	if !ok {
		return errors.New("ConditionalCheckFailedException: customerId does not exist")
	}
	val.InterestDetailsStatus = status
	val.InterestDetailsMessage = message
	if status == "FAILED" || status == "IN_PROGRESS" {
		val.OverallStatus = status
	}
	val.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateAccountOnboarding updates account step in memory.
func (m *MockRequestStatusStore) UpdateAccountOnboarding(ctx context.Context, customerId string, status string, message string, overallStatus string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.status[customerId]
	if !ok {
		return errors.New("ConditionalCheckFailedException: customerId does not exist")
	}
	val.AccountOnboardingStatus = status
	val.AccountOnboardingMessage = message
	val.OverallStatus = overallStatus
	val.UpdatedAt = time.Now().UTC()
	return nil
}

// MockAccountDetailsStore implements AccountDetailsStore in memory.
type MockAccountDetailsStore struct {
	mu      sync.RWMutex
	details map[string]*AccountDetails
}

// NewMockAccountDetailsStore creates a new MockAccountDetailsStore.
func NewMockAccountDetailsStore() *MockAccountDetailsStore {
	return &MockAccountDetailsStore{
		details: make(map[string]*AccountDetails),
	}
}

// GetAccountDetails retrieves details from memory.
func (m *MockAccountDetailsStore) GetAccountDetails(ctx context.Context, customerId string) (*AccountDetails, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.details[customerId]
	if !ok {
		return nil, nil
	}
	cp := *val
	return &cp, nil
}

// SaveAccountDetails saves details in memory.
func (m *MockAccountDetailsStore) SaveAccountDetails(ctx context.Context, details *AccountDetails) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.details[details.CustomerId]; ok {
		return errors.New("ConditionalCheckFailedException: details already exists")
	}
	cp := *details
	cp.CreatedAt = time.Now().UTC()
	m.details[details.CustomerId] = &cp
	return nil
}
