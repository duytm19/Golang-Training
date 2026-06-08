package customer

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/customer-management-service/pkg/customer"
)

// Handler implements customer.ServerInterface
type Handler struct{}

// NewHandler creates a new Handler instance
func NewHandler() *Handler {
	return &Handler{}
}

// HealthCheck (GET /health)
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

// RegisterCustomer (POST /internal/customers/register)
func (h *Handler) RegisterCustomer(c *gin.Context) {
	var req customer.RegisterCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, customer.ErrorResponse{
			Code:          "BAD_REQUEST",
			Message:       err.Error(),
			CorrelationId: c.GetHeader("X-Correlation-Id"),
		})
		return
	}

	correlationID := req.CorrelationId
	if correlationID == "" {
		correlationID = c.GetHeader("X-Correlation-Id")
	}

	switch req.CustomerId {
	case "CUST_FAIL_REGISTER":
		c.JSON(http.StatusInternalServerError, customer.ErrorResponse{
			Code:          "CUST_FAIL_REGISTER",
			Message:       "Simulated downstream customer registration failure",
			CorrelationId: correlationID,
		})
		return
	case "CUST_BAD_REQUEST":
		c.JSON(http.StatusBadRequest, customer.ErrorResponse{
			Code:          "CUST_BAD_REQUEST",
			Message:       "Simulated bad request error",
			CorrelationId: correlationID,
		})
		return
	}

	res := customer.RegisterCustomerResult{
		CustomerId:     req.CustomerId,
		CoreCustomerId: "CORE-" + req.CustomerId,
		Status:         "REGISTERED",
		RegisteredAt:   time.Now().UTC(),
	}

	c.JSON(http.StatusOK, res)
}

// GetCustomer (GET /internal/customers/{customerId})
func (h *Handler) GetCustomer(c *gin.Context, customerId string) {
	correlationID := c.GetHeader("X-Correlation-Id")

	switch customerId {
	case "CUST_FAIL_REGISTER":
		c.JSON(http.StatusInternalServerError, customer.ErrorResponse{
			Code:          "CUST_FAIL_REGISTER",
			Message:       "Simulated customer retrieval failure",
			CorrelationId: correlationID,
		})
		return
	case "CUST_BAD_REQUEST":
		c.JSON(http.StatusBadRequest, customer.ErrorResponse{
			Code:          "CUST_BAD_REQUEST",
			Message:       "Simulated bad request error",
			CorrelationId: correlationID,
		})
		return
	case "CUST_NOT_FOUND":
		c.JSON(http.StatusNotFound, customer.ErrorResponse{
			Code:          "CUST_NOT_FOUND",
			Message:       "Customer not found",
			CorrelationId: correlationID,
		})
		return
	}

	res := customer.RegisterCustomerResult{
		CustomerId:     customerId,
		CoreCustomerId: "CORE-" + customerId,
		Status:         "REGISTERED",
		RegisteredAt:   time.Now().UTC(),
	}

	c.JSON(http.StatusOK, res)
}
