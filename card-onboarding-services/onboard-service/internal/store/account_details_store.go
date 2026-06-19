package store

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// AccountDetails represents the stored business account details.
// Attributes mirror the onboard-service-account-details table and are written
// incrementally as each onboarding step succeeds (see status update rules 12.2/12.4/12.6).
type AccountDetails struct {
	CustomerId       string  `json:"customerId" dynamodbav:"customerId"`
	CoreCustomerId   string  `json:"coreCustomerId,omitempty" dynamodbav:"coreCustomerId,omitempty"`
	CustomerName     string  `json:"customerName,omitempty" dynamodbav:"customerName,omitempty"`
	Email            string  `json:"email,omitempty" dynamodbav:"email,omitempty"`
	ProductCode      string  `json:"productCode,omitempty" dynamodbav:"productCode,omitempty"`
	InterestRate     float64 `json:"interestRate,omitempty" dynamodbav:"interestRate,omitempty"`
	InterestType     string  `json:"interestType,omitempty" dynamodbav:"interestType,omitempty"`
	Currency         string  `json:"currency,omitempty" dynamodbav:"currency,omitempty"`
	AccountId        string  `json:"accountId,omitempty" dynamodbav:"accountId,omitempty"`
	CardId           string  `json:"cardId,omitempty" dynamodbav:"cardId,omitempty"`
	CardType         string  `json:"cardType,omitempty" dynamodbav:"cardType,omitempty"`
	CardNumberMasked string  `json:"cardNumberMasked,omitempty" dynamodbav:"cardNumberMasked,omitempty"`
	CreatedAt        string  `json:"createdAt,omitempty" dynamodbav:"createdAt,omitempty"`
	UpdatedAt        string  `json:"updatedAt,omitempty" dynamodbav:"updatedAt,omitempty"`
}

// AccountDetailsStore defines the persistence operations for account details.
// Writes are step-scoped upserts so each onboarding step only persists the
// attributes it owns, matching the phased status update rules.
type AccountDetailsStore interface {
	GetAccountDetails(ctx context.Context, customerId string) (*AccountDetails, error)
	SaveCustomerInfo(ctx context.Context, customerId, coreCustomerId, customerName, email string) error
	SaveInterestInfo(ctx context.Context, customerId, productCode string, interestRate float64, interestType, currency string) error
	SaveCardInfo(ctx context.Context, customerId, accountId, cardId, cardType, cardNumberMasked string) error
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

// SaveCustomerInfo upserts the customer-level attributes (step 12.2).
func (s *DynamoAccountDetailsStore) SaveCustomerInfo(ctx context.Context, customerId, coreCustomerId, customerName, email string) error {
	return s.upsert(ctx, customerId,
		"SET coreCustomerId = :coreId, customerName = :name, email = :email",
		map[string]types.AttributeValue{
			":coreId": &types.AttributeValueMemberS{Value: coreCustomerId},
			":name":   &types.AttributeValueMemberS{Value: customerName},
			":email":  &types.AttributeValueMemberS{Value: email},
		})
}

// SaveInterestInfo upserts the interest/product attributes (step 12.4).
func (s *DynamoAccountDetailsStore) SaveInterestInfo(ctx context.Context, customerId, productCode string, interestRate float64, interestType, currency string) error {
	return s.upsert(ctx, customerId,
		"SET productCode = :pc, interestRate = :rate, interestType = :it, currency = :cur",
		map[string]types.AttributeValue{
			":pc":   &types.AttributeValueMemberS{Value: productCode},
			":rate": &types.AttributeValueMemberN{Value: strconv.FormatFloat(interestRate, 'f', -1, 64)},
			":it":   &types.AttributeValueMemberS{Value: interestType},
			":cur":  &types.AttributeValueMemberS{Value: currency},
		})
}

// SaveCardInfo upserts the account/card attributes (step 12.6).
func (s *DynamoAccountDetailsStore) SaveCardInfo(ctx context.Context, customerId, accountId, cardId, cardType, cardNumberMasked string) error {
	return s.upsert(ctx, customerId,
		"SET accountId = :aid, cardId = :cid, cardType = :ct, cardNumberMasked = :masked",
		map[string]types.AttributeValue{
			":aid":    &types.AttributeValueMemberS{Value: accountId},
			":cid":    &types.AttributeValueMemberS{Value: cardId},
			":ct":     &types.AttributeValueMemberS{Value: cardType},
			":masked": &types.AttributeValueMemberS{Value: cardNumberMasked},
		})
}

// upsert applies a step-scoped SET expression while stamping createdAt once and
// updatedAt on every write. UpdateItem creates the item if it does not yet exist,
// so repeated (resumed) calls are idempotent.
func (s *DynamoAccountDetailsStore) upsert(ctx context.Context, customerId, setExpr string, values map[string]types.AttributeValue) error {
	now := time.Now().UTC().Format(time.RFC3339)
	values[":now"] = &types.AttributeValueMemberS{Value: now}

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(s.tableName),
		Key:                       map[string]types.AttributeValue{"customerId": &types.AttributeValueMemberS{Value: customerId}},
		UpdateExpression:          aws.String(setExpr + ", createdAt = if_not_exists(createdAt, :now), updatedAt = :now"),
		ExpressionAttributeValues: values,
	})
	return err
}
