package account

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/account-management-service/pkg/account"
)

// Handler implements account.ServerInterface
type Handler struct{}

// NewHandler creates a new Handler instance
func NewHandler() *Handler {
	return &Handler{}
}

// HealthCheck (GET /health)
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

// GetInterestDetails (GET /internal/accounts/{customerId}/interest-details)
func (h *Handler) GetInterestDetails(c *gin.Context, customerId string) {
	correlationID := c.GetHeader("X-Correlation-Id")
	if correlationID == "" {
		correlationID = c.GetHeader("Correlation-Id")
	}

	switch customerId {
	case "CUST_FAIL_INTEREST":
		c.JSON(http.StatusInternalServerError, account.ErrorResponse{
			Code:          "CUST_FAIL_INTEREST",
			Message:       "Simulated downstream interest rate retrieval failure",
			CorrelationId: correlationID,
		})
		return
	case "CUST_NO_INTEREST":
		c.JSON(http.StatusNotFound, account.ErrorResponse{
			Code:          "CUST_NO_INTEREST",
			Message:       "Simulated interest rate not found",
			CorrelationId: correlationID,
		})
		return
	}

	res := account.InterestDetailsResponse{
		CustomerId:   customerId,
		ProductCode:  "SAVINGS_BASIC",
		InterestRate: 4.5,
		InterestType: account.VARIABLE,
		Currency:     "AUD",
	}

	c.JSON(http.StatusOK, res)
}
