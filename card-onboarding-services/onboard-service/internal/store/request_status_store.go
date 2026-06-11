package store

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// RequestStatus represents the stored state of the onboarding workflow.
type RequestStatus struct {
	CustomerId                  string    `json:"customerId" dynamodbav:"customerId"`
	JobId                       string    `json:"jobId" dynamodbav:"jobId"`
	RecordId                    string    `json:"recordId" dynamodbav:"recordId"`
	OverallStatus               string    `json:"overallStatus" dynamodbav:"overallStatus"`
	CoreCustomerId              string    `json:"coreCustomerId,omitempty" dynamodbav:"coreCustomerId,omitempty"`
	CustomerRegistrationStatus  string    `json:"customerRegistrationStatus,omitempty" dynamodbav:"customerRegistrationStatus,omitempty"`
	CustomerRegistrationMessage string    `json:"customerRegistrationMessage,omitempty" dynamodbav:"customerRegistrationMessage,omitempty"`
	InterestDetailsStatus       string    `json:"interestDetailsStatus,omitempty" dynamodbav:"interestDetailsStatus,omitempty"`
	InterestDetailsMessage      string    `json:"interestDetailsMessage,omitempty" dynamodbav:"interestDetailsMessage,omitempty"`
	AccountOnboardingStatus     string    `json:"accountOnboardingStatus,omitempty" dynamodbav:"accountOnboardingStatus,omitempty"`
	AccountOnboardingMessage    string    `json:"accountOnboardingMessage,omitempty" dynamodbav:"accountOnboardingMessage,omitempty"`
	UpdatedAt                   time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
}

// RequestStatusStore defines the persistence operations for request status tracking.
type RequestStatusStore interface {
	GetRequestStatus(ctx context.Context, customerId string) (*RequestStatus, error)
	CreateRequestStatus(ctx context.Context, customerId, jobId, recordId string) error
	UpdateCustomerRegistration(ctx context.Context, customerId string, status string, coreCustomerId string, message string) error
	UpdateInterestDetails(ctx context.Context, customerId string, status string, message string) error
	UpdateAccountOnboarding(ctx context.Context, customerId string, status string, message string, overallStatus string) error
}

// DynamoRequestStatusStore implements RequestStatusStore using DynamoDB.
type DynamoRequestStatusStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoRequestStatusStore creates a new DynamoRequestStatusStore instance.
func NewDynamoRequestStatusStore(client *dynamodb.Client, tableName string) *DynamoRequestStatusStore {
	return &DynamoRequestStatusStore{
		client:    client,
		tableName: tableName,
	}
}

// GetRequestStatus retrieves workflow status by customerId.
func (s *DynamoRequestStatusStore) GetRequestStatus(ctx context.Context, customerId string) (*RequestStatus, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"customerId": &types.AttributeValueMemberS{Value: customerId},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Item) == 0 {
		return nil, nil
	}

	var res RequestStatus
	if err := attributevalue.UnmarshalMap(out.Item, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// CreateRequestStatus initializes a workflow tracking record with safety checks.
func (s *DynamoRequestStatusStore) CreateRequestStatus(ctx context.Context, customerId, jobId, recordId string) error {
	now := time.Now().UTC()
	item := map[string]types.AttributeValue{
		"customerId":                 &types.AttributeValueMemberS{Value: customerId},
		"jobId":                      &types.AttributeValueMemberS{Value: jobId},
		"recordId":                   &types.AttributeValueMemberS{Value: recordId},
		"overallStatus":              &types.AttributeValueMemberS{Value: "IN_PROGRESS"},
		"customerRegistrationStatus": &types.AttributeValueMemberS{Value: "IN_PROGRESS"},
		"updatedAt":                  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(customerId)"),
	})
	return err
}

// UpdateCustomerRegistration updates customer step state.
func (s *DynamoRequestStatusStore) UpdateCustomerRegistration(ctx context.Context, customerId string, status string, coreCustomerId string, message string) error {
	now := time.Now().UTC()
	updateExpr := "SET customerRegistrationStatus = :status, customerRegistrationMessage = :msg, updatedAt = :updatedAt"
	exprValues := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: status},
		":msg":       &types.AttributeValueMemberS{Value: message},
		":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	if coreCustomerId != "" {
		updateExpr += ", coreCustomerId = :coreId"
		exprValues[":coreId"] = &types.AttributeValueMemberS{Value: coreCustomerId}
	}

	if status == "FAILED" || status == "IN_PROGRESS" {
		updateExpr += ", overallStatus = :overall"
		exprValues[":overall"] = &types.AttributeValueMemberS{Value: status}
	}

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(s.tableName),
		Key:                       map[string]types.AttributeValue{"customerId": &types.AttributeValueMemberS{Value: customerId}},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeValues: exprValues,
		ConditionExpression:       aws.String("attribute_exists(customerId)"),
	})
	return err
}

// UpdateInterestDetails updates interest details fetching step state.
func (s *DynamoRequestStatusStore) UpdateInterestDetails(ctx context.Context, customerId string, status string, message string) error {
	now := time.Now().UTC()
	updateExpr := "SET interestDetailsStatus = :status, interestDetailsMessage = :msg, updatedAt = :updatedAt"
	exprValues := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: status},
		":msg":       &types.AttributeValueMemberS{Value: message},
		":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	if status == "FAILED" || status == "IN_PROGRESS" {
		updateExpr += ", overallStatus = :overall"
		exprValues[":overall"] = &types.AttributeValueMemberS{Value: status}
	}

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(s.tableName),
		Key:                       map[string]types.AttributeValue{"customerId": &types.AttributeValueMemberS{Value: customerId}},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeValues: exprValues,
		ConditionExpression:       aws.String("attribute_exists(customerId)"),
	})
	return err
}

// UpdateAccountOnboarding updates the final account onboarding step state and overall status.
func (s *DynamoRequestStatusStore) UpdateAccountOnboarding(ctx context.Context, customerId string, status string, message string, overallStatus string) error {
	now := time.Now().UTC()
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key:       map[string]types.AttributeValue{"customerId": &types.AttributeValueMemberS{Value: customerId}},
		UpdateExpression: aws.String("SET accountOnboardingStatus = :status, accountOnboardingMessage = :msg, overallStatus = :overall, updatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberS{Value: status},
			":msg":       &types.AttributeValueMemberS{Value: message},
			":overall":   &types.AttributeValueMemberS{Value: overallStatus},
			":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(customerId)"),
	})
	return err
}
