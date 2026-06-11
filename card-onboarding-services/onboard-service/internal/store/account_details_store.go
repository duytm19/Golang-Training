package store

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// AccountDetails represents the stored onboarding result.
type AccountDetails struct {
	CustomerId     string    `json:"customerId" dynamodbav:"customerId"`
	CoreCustomerId string    `json:"coreCustomerId" dynamodbav:"coreCustomerId"`
	AccountId      string    `json:"accountId" dynamodbav:"accountId"`
	CardId         string    `json:"cardId" dynamodbav:"cardId"`
	Status         string    `json:"status" dynamodbav:"status"`
	CreatedAt      time.Time `json:"createdAt" dynamodbav:"createdAt"`
}

// AccountDetailsStore defines the persistence operations for finalized account details.
type AccountDetailsStore interface {
	GetAccountDetails(ctx context.Context, customerId string) (*AccountDetails, error)
	SaveAccountDetails(ctx context.Context, details *AccountDetails) error
}

// DynamoAccountDetailsStore implements AccountDetailsStore using DynamoDB.
type DynamoAccountDetailsStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoAccountDetailsStore creates a new DynamoAccountDetailsStore instance.
func NewDynamoAccountDetailsStore(client *dynamodb.Client, tableName string) *DynamoAccountDetailsStore {
	return &DynamoAccountDetailsStore{
		client:    client,
		tableName: tableName,
	}
}

// GetAccountDetails retrieves account mappings by customerId.
func (s *DynamoAccountDetailsStore) GetAccountDetails(ctx context.Context, customerId string) (*AccountDetails, error) {
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

	var res AccountDetails
	if err := attributevalue.UnmarshalMap(out.Item, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SaveAccountDetails saves finalized account mappings with an overwrite guard.
func (s *DynamoAccountDetailsStore) SaveAccountDetails(ctx context.Context, details *AccountDetails) error {
	details.CreatedAt = time.Now().UTC()
	item, err := attributevalue.MarshalMap(details)
	if err != nil {
		return err
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(customerId)"),
	})
	return err
}
