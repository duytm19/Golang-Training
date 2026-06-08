package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/account-management-service/internal/account"
	accountpkg "github.com/org/card-onboarding-services/account-management-service/pkg/account"
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CorrelationAndLoggingMiddleware())

	h := account.NewHandler()
	accountpkg.RegisterHandlers(r, h)

	log.Printf(`{"level":"info","message":"Starting account-management-service on port %s"}`+"\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf(`{"level":"error","message":"Failed to run server: %v"}`+"\n", err)
	}
}
