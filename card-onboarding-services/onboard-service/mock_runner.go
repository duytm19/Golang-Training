package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/onboard-service/internal/card"
	"github.com/org/card-onboarding-services/onboard-service/internal/client"
	"github.com/org/card-onboarding-services/onboard-service/internal/orchestration"
	"github.com/org/card-onboarding-services/onboard-service/internal/store"
	onboardpkg "github.com/org/card-onboarding-services/onboard-service/pkg/onboard"
)

func CorrelationAndLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		corrID := c.GetHeader("X-Correlation-Id")
		if corrID == "" {
			corrID = c.GetHeader("Correlation-Id")
		}
		c.Set("correlationId", corrID)

		if corrID != "" {
			c.Header("X-Correlation-Id", corrID)
		}

		c.Next()

		log.Printf(`{"level":"info","correlationId":"%s","status":%d,"method":"%s","path":"%s"}`+"\n",
			corrID, c.Writer.Status(), c.Request.Method, c.Request.URL.Path)
	}
}

func main() {
	// Use mock stores
	statusStore := store.NewMockRequestStatusStore()
	detailsStore := store.NewMockAccountDetailsStore()

	// Connect to customer & account mocks on local ports
	custClient := client.NewHttpCustomerClient("http://localhost:8081")
	acctClient := client.NewHttpAccountClient("http://localhost:8082")

	orchestrator := orchestration.NewOrchestrationService(statusStore, detailsStore, custClient, acctClient)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CorrelationAndLoggingMiddleware())

	h := card.NewHandler(orchestrator)
	onboardpkg.RegisterHandlers(r, h)

	log.Println(`{"level":"info","message":"Starting MOCK onboard-service on port 8080"}`)
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf(`{"level":"error","message":"Failed to run mock server: %v"}`, err)
	}
}
