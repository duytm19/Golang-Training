package orchestration

import (
	"context"
	"errors"
	"fmt"

	"github.com/org/card-onboarding-services/onboard-service/internal/client"
	"github.com/org/card-onboarding-services/onboard-service/internal/store"
	"github.com/org/card-onboarding-services/onboard-service/pkg/onboard"
)

// OrchestrationService manages the onboarding state machine and its transition steps.
type OrchestrationService struct {
	statusStore  store.RequestStatusStore
	detailsStore store.AccountDetailsStore
	custClient   client.CustomerClient
	acctClient   client.AccountClient
}

// NewOrchestrationService initializes a new OrchestrationService.
func NewOrchestrationService(
	statusStore store.RequestStatusStore,
	detailsStore store.AccountDetailsStore,
	custClient client.CustomerClient,
	acctClient client.AccountClient,
) *OrchestrationService {
	return &OrchestrationService{
		statusStore:  statusStore,
		detailsStore: detailsStore,
		custClient:   custClient,
		acctClient:   acctClient,
	}
}

// GetStatus retrieves the current onboarding workflow status.
func (s *OrchestrationService) GetStatus(ctx context.Context, customerId string) (*store.RequestStatus, error) {
	return s.statusStore.GetRequestStatus(ctx, customerId)
}

// Onboard handles onboarding requests using a step-by-step resume state machine.
func (s *OrchestrationService) Onboard(ctx context.Context, req onboard.OnboardRequest) (*onboard.OnboardResponse, error) {
	customerId := req.CustomerId
	correlationID := req.CorrelationId

	// Check existing workflow status
	status, err := s.statusStore.GetRequestStatus(ctx, customerId)
	if err != nil {
		return nil, fmt.Errorf("failed to get request status: %w", err)
	}

	// Case E: Already Completed. Return immediately.
	if status != nil && status.OverallStatus == "SUCCEEDED" {
		details, err := s.detailsStore.GetAccountDetails(ctx, customerId)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve account details: %w", err)
		}
		if details == nil {
			return nil, errors.New("request status overallStatus is SUCCEEDED but account details were not found")
		}
		return &onboard.OnboardResponse{
			CustomerId:     details.CustomerId,
			CoreCustomerId: details.CoreCustomerId,
			AccountId:      details.AccountId,
			CardId:         details.CardId,
			Status:         details.Status,
		}, nil
	}

	// Case A: No existing status. Initialize it.
	if status == nil {
		err = s.statusStore.CreateRequestStatus(ctx, customerId, req.JobId, req.RecordId)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize request status: %w", err)
		}
	}

	// Refresh status context
	status, err = s.statusStore.GetRequestStatus(ctx, customerId)
	if err != nil {
		return nil, fmt.Errorf("failed to reload request status: %w", err)
	}

	var coreCustomerId = status.CoreCustomerId

	// Step 1: Customer Registration (Case B / Start)
	if status.CustomerRegistrationStatus != "SUCCEEDED" {
		if status.CustomerRegistrationStatus == "FAILED" {
			err = s.statusStore.UpdateCustomerRegistration(ctx, customerId, "IN_PROGRESS", "", "Retrying customer registration")
			if err != nil {
				return nil, err
			}
		}

		custRes, err := s.custClient.RegisterCustomer(ctx, correlationID, customerId, req.HolderName, string(req.Email))
		if err != nil {
			_ = s.statusStore.UpdateCustomerRegistration(ctx, customerId, "FAILED", "", err.Error())
			return nil, fmt.Errorf("customer registration step failed: %w", err)
		}

		coreCustomerId = custRes.CoreCustomerId
		err = s.statusStore.UpdateCustomerRegistration(ctx, customerId, "SUCCEEDED", coreCustomerId, "Customer registered successfully")
		if err != nil {
			return nil, err
		}
	}

	// Step 2: Interest Details Fetching (Case C)
	if status.InterestDetailsStatus != "SUCCEEDED" {
		err = s.statusStore.UpdateInterestDetails(ctx, customerId, "IN_PROGRESS", "Fetching account interest details")
		if err != nil {
			return nil, err
		}

		_, err = s.acctClient.GetInterestDetails(ctx, correlationID, customerId)
		if err != nil {
			_ = s.statusStore.UpdateInterestDetails(ctx, customerId, "FAILED", err.Error())
			return nil, fmt.Errorf("interest details step failed: %w", err)
		}

		err = s.statusStore.UpdateInterestDetails(ctx, customerId, "SUCCEEDED", "Interest details fetched successfully")
		if err != nil {
			return nil, err
		}
	}

	// Step 3: Account Onboarding / Saving Details (Case D)
	if status.AccountOnboardingStatus != "SUCCEEDED" {
		err = s.statusStore.UpdateAccountOnboarding(ctx, customerId, "IN_PROGRESS", "Saving onboarding account details", "IN_PROGRESS")
		if err != nil {
			return nil, err
		}

		accountId := "ACC-" + customerId
		cardId := "CARD-" + customerId + "-001"

		details := &store.AccountDetails{
			CustomerId:     customerId,
			CoreCustomerId: coreCustomerId,
			AccountId:      accountId,
			CardId:         cardId,
			Status:         "ONBOARDED",
		}

		err = s.detailsStore.SaveAccountDetails(ctx, details)
		if err != nil {
			_ = s.statusStore.UpdateAccountOnboarding(ctx, customerId, "FAILED", err.Error(), "FAILED")
			return nil, fmt.Errorf("account details saving step failed: %w", err)
		}

		err = s.statusStore.UpdateAccountOnboarding(ctx, customerId, "SUCCEEDED", "Account onboarding completed successfully", "SUCCEEDED")
		if err != nil {
			return nil, err
		}
	}

	return &onboard.OnboardResponse{
		CustomerId:     customerId,
		CoreCustomerId: coreCustomerId,
		AccountId:      "ACC-" + customerId,
		CardId:         "CARD-" + customerId + "-001",
		Status:         "ONBOARDED",
	}, nil
}
