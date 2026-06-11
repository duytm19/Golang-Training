package main

import (
	"context"
	"log"
	"net/http"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/onboard-service/internal/card"
	"github.com/org/card-onboarding-services/onboard-service/internal/client"
	"github.com/org/card-onboarding-services/onboard-service/internal/config"
	"github.com/org/card-onboarding-services/onboard-service/internal/orchestration"
	"github.com/org/card-onboarding-services/onboard-service/internal/store"
	onboardpkg "github.com/org/card-onboarding-services/onboard-service/pkg/onboard"
)

// CorrelationAndLoggingMiddleware extracts and propagates correlation ID and logs incoming HTTP requests.
func CorrelationAndLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		corrID := c.GetHeader("X-Correlation-Id")
		if corrID == "" {
			corrID = c.GetHeader("Correlation-Id")
		}
		c.Set("correlationId", corrID)

		// Set header in response so it's always tracked back
		if corrID != "" {
			c.Header("X-Correlation-Id", corrID)
		}

		c.Next()

		log.Printf(`{"level":"info","correlationId":"%s","status":%d,"method":"%s","path":"%s"}`+"\n",
			corrID, c.Writer.Status(), c.Request.Method, c.Request.URL.Path)
	}
}

func main() {
	appConfig := config.Load()

	// Initialize AWS Config for DynamoDB
	// LoadDefaultConfig will attempt to resolve AWS credentials and config.
	// In local testing/mock run, we can mock it, but for main service startup, we attempt loading.
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(appConfig.AwsRegion))
	if err != nil {
		log.Printf(`{"level":"warn","message":"Failed to load default AWS config, DynamoDB client might not work: %v"}`+"\n", err)
	}

	dbClient := dynamodb.NewFromConfig(awsCfg)

	// Initialize stores
	statusStore := store.NewDynamoRequestStatusStore(dbClient, appConfig.RequestStatusTable)
	detailsStore := store.NewDynamoAccountDetailsStore(dbClient, appConfig.AccountDetailsTable)

	// Initialize downstream clients
	custClient := client.NewHttpCustomerClient(appConfig.CustomerServiceURL)
	acctClient := client.NewHttpAccountClient(appConfig.AccountServiceURL)

	// Initialize orchestration service
	orchestrator := orchestration.NewOrchestrationService(statusStore, detailsStore, custClient, acctClient)

	// Initialize server routing
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CorrelationAndLoggingMiddleware())

	h := card.NewHandler(orchestrator)
	onboardpkg.RegisterHandlers(r, h)

	log.Printf(`{"level":"info","message":"Starting onboard-service on port %s"}`+"\n", appConfig.Port)
	if err := http.ListenAndServe(":"+appConfig.Port, r); err != nil {
		log.Fatalf(`{"level":"error","message":"Failed to run onboard-service: %v"}`+"\n", err)
	}
}
