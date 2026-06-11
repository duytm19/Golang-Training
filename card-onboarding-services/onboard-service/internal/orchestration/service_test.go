package orchestration

import (
	"context"
	"testing"
	"time"

	accountpkg "github.com/org/card-onboarding-services/account-management-service/pkg/account"
	customerpkg "github.com/org/card-onboarding-services/customer-management-service/pkg/customer"
	"github.com/org/card-onboarding-services/onboard-service/internal/store"
	"github.com/org/card-onboarding-services/onboard-service/pkg/onboard"
)

type MockCustomerClient struct {
	RegisterCalls int
	RegisterFunc  func(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customerpkg.RegisterCustomerResult, error)
	GetCalls      int
	GetFunc       func(ctx context.Context, correlationID string, customerID string) (*customerpkg.RegisterCustomerResult, error)
}

func (m *MockCustomerClient) RegisterCustomer(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customerpkg.RegisterCustomerResult, error) {
	m.RegisterCalls++
	return m.RegisterFunc(ctx, correlationID, customerID, holderName, email)
}

func (m *MockCustomerClient) GetCustomer(ctx context.Context, correlationID string, customerID string) (*customerpkg.RegisterCustomerResult, error) {
	m.GetCalls++
	return m.GetFunc(ctx, correlationID, customerID)
}

type MockAccountClient struct {
	GetInterestCalls int
	GetInterestFunc  func(ctx context.Context, correlationID string, customerID string) (*accountpkg.InterestDetailsResponse, error)
}

func (m *MockAccountClient) GetInterestDetails(ctx context.Context, correlationID string, customerID string) (*accountpkg.InterestDetailsResponse, error) {
	m.GetInterestCalls++
	return m.GetInterestFunc(ctx, correlationID, customerID)
}

func TestOnboard_HappyPath(t *testing.T) {
	statusStore := store.NewMockRequestStatusStore()
	detailsStore := store.NewMockAccountDetailsStore()

	custClient := &MockCustomerClient{
		RegisterFunc: func(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customerpkg.RegisterCustomerResult, error) {
			return &customerpkg.RegisterCustomerResult{
				CustomerId:     customerID,
				CoreCustomerId: "CORE-" + customerID,
				Status:         "REGISTERED",
				RegisteredAt:   time.Now(),
			}, nil
		},
	}

	acctClient := &MockAccountClient{
		GetInterestFunc: func(ctx context.Context, correlationID string, customerID string) (*accountpkg.InterestDetailsResponse, error) {
			return &accountpkg.InterestDetailsResponse{
				CustomerId:   customerID,
				ProductCode:  "SAVINGS_BASIC",
				InterestRate: 4.5,
				InterestType: accountpkg.VARIABLE,
				Currency:     "AUD",
			}, nil
		},
	}

	svc := NewOrchestrationService(statusStore, detailsStore, custClient, acctClient)

	req := onboard.OnboardRequest{
		CustomerId:    "CUST001",
		CorrelationId: "corr-123",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
		JobId:         "job-1",
		RecordId:      "rec-1",
	}

	res, err := svc.Onboard(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if res.CustomerId != "CUST001" {
		t.Errorf("Expected CustomerId CUST001, got %s", res.CustomerId)
	}
	if res.CoreCustomerId != "CORE-CUST001" {
		t.Errorf("Expected CoreCustomerId CORE-CUST001, got %s", res.CoreCustomerId)
	}
	if res.AccountId != "ACC-CUST001" {
		t.Errorf("Expected AccountId ACC-CUST001, got %s", res.AccountId)
	}
	if res.CardId != "CARD-CUST001-001" {
		t.Errorf("Expected CardId CARD-CUST001-001, got %s", res.CardId)
	}

	// Verify database state
	status, err := statusStore.GetRequestStatus(context.Background(), "CUST001")
	if err != nil || status == nil {
		t.Fatalf("Failed to fetch status: %v", err)
	}
	if status.OverallStatus != "SUCCEEDED" {
		t.Errorf("Expected overallStatus SUCCEEDED, got %s", status.OverallStatus)
	}
	if status.CustomerRegistrationStatus != "SUCCEEDED" {
		t.Errorf("Expected customerRegistrationStatus SUCCEEDED, got %s", status.CustomerRegistrationStatus)
	}
	if status.InterestDetailsStatus != "SUCCEEDED" {
		t.Errorf("Expected interestDetailsStatus SUCCEEDED, got %s", status.InterestDetailsStatus)
	}
	if status.AccountOnboardingStatus != "SUCCEEDED" {
		t.Errorf("Expected accountOnboardingStatus SUCCEEDED, got %s", status.AccountOnboardingStatus)
	}

	details, err := detailsStore.GetAccountDetails(context.Background(), "CUST001")
	if err != nil || details == nil {
		t.Fatalf("Failed to fetch details: %v", err)
	}
	if details.AccountId != "ACC-CUST001" {
		t.Errorf("Expected saved AccountId ACC-CUST001, got %s", details.AccountId)
	}
}

func TestOnboard_ResumeFromStep2(t *testing.T) {
	statusStore := store.NewMockRequestStatusStore()
	detailsStore := store.NewMockAccountDetailsStore()

	// Pre-populate status: Step 1 Succeeded, but overall is FAILED
	// Since UpdateCustomerRegistration updates state, let's create and set it manually in mock store map
	// Wait, we can just use the store's helper methods:
	_ = statusStore.CreateRequestStatus(context.Background(), "CUST001", "job-1", "rec-1")
	_ = statusStore.UpdateCustomerRegistration(context.Background(), "CUST001", "SUCCEEDED", "CORE-CUST001", "Customer registered successfully")
	_ = statusStore.UpdateInterestDetails(context.Background(), "CUST001", "FAILED", "Downstream error")

	custClient := &MockCustomerClient{
		RegisterFunc: func(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customerpkg.RegisterCustomerResult, error) {
			t.Fatal("RegisterCustomer should NOT be called on resume from Step 2")
			return nil, nil
		},
	}

	acctClient := &MockAccountClient{
		GetInterestFunc: func(ctx context.Context, correlationID string, customerID string) (*accountpkg.InterestDetailsResponse, error) {
			return &accountpkg.InterestDetailsResponse{
				CustomerId:   customerID,
				ProductCode:  "SAVINGS_BASIC",
				InterestRate: 4.5,
				InterestType: accountpkg.VARIABLE,
				Currency:     "AUD",
			}, nil
		},
	}

	svc := NewOrchestrationService(statusStore, detailsStore, custClient, acctClient)

	req := onboard.OnboardRequest{
		CustomerId:    "CUST001",
		CorrelationId: "corr-123",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
		JobId:         "job-1",
		RecordId:      "rec-1",
	}

	_, err := svc.Onboard(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if custClient.RegisterCalls != 0 {
		t.Errorf("RegisterCustomer called %d times, expected 0", custClient.RegisterCalls)
	}
	if acctClient.GetInterestCalls != 1 {
		t.Errorf("GetInterestDetails called %d times, expected 1", acctClient.GetInterestCalls)
	}

	// Verify state
	status, _ := statusStore.GetRequestStatus(context.Background(), "CUST001")
	if status.OverallStatus != "SUCCEEDED" {
		t.Errorf("Expected overallStatus SUCCEEDED, got %s", status.OverallStatus)
	}
}

func TestOnboard_ResumeFromStep3(t *testing.T) {
	statusStore := store.NewMockRequestStatusStore()
	detailsStore := store.NewMockAccountDetailsStore()

	_ = statusStore.CreateRequestStatus(context.Background(), "CUST001", "job-1", "rec-1")
	_ = statusStore.UpdateCustomerRegistration(context.Background(), "CUST001", "SUCCEEDED", "CORE-CUST001", "Customer registered successfully")
	_ = statusStore.UpdateInterestDetails(context.Background(), "CUST001", "SUCCEEDED", "Interest details fetched")
	_ = statusStore.UpdateAccountOnboarding(context.Background(), "CUST001", "FAILED", "Save failed", "FAILED")

	custClient := &MockCustomerClient{
		RegisterFunc: func(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customerpkg.RegisterCustomerResult, error) {
			t.Fatal("RegisterCustomer should NOT be called on resume from Step 3")
			return nil, nil
		},
	}

	acctClient := &MockAccountClient{
		GetInterestFunc: func(ctx context.Context, correlationID string, customerID string) (*accountpkg.InterestDetailsResponse, error) {
			t.Fatal("GetInterestDetails should NOT be called on resume from Step 3")
			return nil, nil
		},
	}

	svc := NewOrchestrationService(statusStore, detailsStore, custClient, acctClient)

	req := onboard.OnboardRequest{
		CustomerId:    "CUST001",
		CorrelationId: "corr-123",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
		JobId:         "job-1",
		RecordId:      "rec-1",
	}

	_, err := svc.Onboard(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if custClient.RegisterCalls != 0 || acctClient.GetInterestCalls != 0 {
		t.Errorf("Downstream calls made when resuming from save step")
	}

	// Verify state
	status, _ := statusStore.GetRequestStatus(context.Background(), "CUST001")
	if status.OverallStatus != "SUCCEEDED" {
		t.Errorf("Expected overallStatus SUCCEEDED, got %s", status.OverallStatus)
	}
}

func TestOnboard_Idempotency(t *testing.T) {
	statusStore := store.NewMockRequestStatusStore()
	detailsStore := store.NewMockAccountDetailsStore()

	// Pre-populate overallStatus SUCCEEDED
	_ = statusStore.CreateRequestStatus(context.Background(), "CUST001", "job-1", "rec-1")
	_ = statusStore.UpdateCustomerRegistration(context.Background(), "CUST001", "SUCCEEDED", "CORE-CUST001", "Customer registered successfully")
	_ = statusStore.UpdateInterestDetails(context.Background(), "CUST001", "SUCCEEDED", "Interest details fetched")
	_ = statusStore.UpdateAccountOnboarding(context.Background(), "CUST001", "SUCCEEDED", "Account onboarding completed", "SUCCEEDED")

	_ = detailsStore.SaveAccountDetails(context.Background(), &store.AccountDetails{
		CustomerId:     "CUST001",
		CoreCustomerId: "CORE-CUST001",
		AccountId:      "ACC-CUST001",
		CardId:         "CARD-CUST001-001",
		Status:         "ONBOARDED",
	})

	custClient := &MockCustomerClient{
		RegisterFunc: func(ctx context.Context, correlationID string, customerID string, holderName string, email string) (*customerpkg.RegisterCustomerResult, error) {
			t.Fatal("RegisterCustomer should NOT be called on idempotent return")
			return nil, nil
		},
	}

	acctClient := &MockAccountClient{
		GetInterestFunc: func(ctx context.Context, correlationID string, customerID string) (*accountpkg.InterestDetailsResponse, error) {
			t.Fatal("GetInterestDetails should NOT be called on idempotent return")
			return nil, nil
		},
	}

	svc := NewOrchestrationService(statusStore, detailsStore, custClient, acctClient)

	req := onboard.OnboardRequest{
		CustomerId:    "CUST001",
		CorrelationId: "corr-123",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
		JobId:         "job-1",
		RecordId:      "rec-1",
	}

	res, err := svc.Onboard(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if res.AccountId != "ACC-CUST001" || res.CardId != "CARD-CUST001-001" {
		t.Errorf("Expected saved details returned, got %v", res)
	}
}
