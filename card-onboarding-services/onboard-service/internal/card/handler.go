package card

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/onboard-service/internal/appmetrics"
	"github.com/org/card-onboarding-services/onboard-service/internal/orchestration"
	"github.com/org/card-onboarding-services/onboard-service/pkg/onboard"
)

// Handler implements onboard.ServerInterface
type Handler struct {
	orchestrator *orchestration.OrchestrationService
}

// NewHandler creates a new Handler instance.
func NewHandler(orchestrator *orchestration.OrchestrationService) *Handler {
	return &Handler{
		orchestrator: orchestrator,
	}
}

// HealthCheck (GET /health)
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

// OnboardCard (POST /internal/cards/onboard)
func (h *Handler) OnboardCard(c *gin.Context) {
	start := time.Now()
	var req onboard.OnboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, onboard.ErrorResponse{
			Code:          "BAD_REQUEST",
			Message:       err.Error(),
			CorrelationId: c.GetHeader("X-Correlation-Id"),
		})
		appmetrics.EmitBusinessValidationFailed([]string{"error_type:bad_request"})
		return
	}

	correlationID := req.CorrelationId
	if correlationID == "" {
		correlationID = c.GetHeader("X-Correlation-Id")
	}

	resp, err := h.orchestrator.Onboard(c.Request.Context(), req)
	duration := time.Since(start)
	durationMs := duration.Milliseconds()

	if err != nil {
		log.Printf(`{"level":"error","message":"Onboarding request failed","correlationId":"%s","jobId":"%s","customerId":"%s","durationMs":%d,"error":"%s"}`+"\n",
			correlationID, req.JobId, req.CustomerId, durationMs, err.Error())
		c.JSON(http.StatusInternalServerError, onboard.ErrorResponse{
			Code:          "ONBOARDING_FAILURE",
			Message:       err.Error(),
			CorrelationId: correlationID,
		})
		appmetrics.EmitProcessingDuration(duration, []string{"status:failed"})
		return
	}

	log.Printf(`{"level":"info","message":"Onboarding request processed","correlationId":"%s","jobId":"%s","customerId":"%s","durationMs":%d}`+"\n",
		correlationID, req.JobId, req.CustomerId, durationMs)

	appmetrics.EmitRecordAccepted([]string{"status:success"})
	appmetrics.EmitProcessingDuration(duration, []string{"status:success"})

	c.JSON(http.StatusOK, resp)
}

// GetOnboardStatus (GET /internal/cards/{customerId}/status)
func (h *Handler) GetOnboardStatus(c *gin.Context, customerId string) {
	correlationID := c.GetHeader("X-Correlation-Id")

	status, err := h.orchestrator.GetStatus(c.Request.Context(), customerId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, onboard.ErrorResponse{
			Code:          "DATABASE_ERROR",
			Message:       err.Error(),
			CorrelationId: correlationID,
		})
		return
	}

	if status == nil {
		c.JSON(http.StatusNotFound, onboard.ErrorResponse{
			Code:          "NOT_FOUND",
			Message:       "onboarding status not found for customerID: " + customerId,
			CorrelationId: correlationID,
		})
		return
	}

	res := onboard.StatusResponse{
		CustomerId:                  status.CustomerId,
		OverallStatus:               onboard.StatusResponseOverallStatus(status.OverallStatus),
		CustomerRegistrationStatus:  customerRegStatusPtr(status.CustomerRegistrationStatus),
		CustomerRegistrationMessage: strToPtr(status.CustomerRegistrationMessage),
		InterestDetailsStatus:       interestStatusPtr(status.InterestDetailsStatus),
		InterestDetailsMessage:      strToPtr(status.InterestDetailsMessage),
		AccountOnboardingStatus:     accountStatusPtr(status.AccountOnboardingStatus),
		AccountOnboardingMessage:    strToPtr(status.AccountOnboardingMessage),
	}

	c.JSON(http.StatusOK, res)
}

func strToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func customerRegStatusPtr(s string) *onboard.StatusResponseCustomerRegistrationStatus {
	if s == "" {
		return nil
	}
	v := onboard.StatusResponseCustomerRegistrationStatus(s)
	return &v
}

func interestStatusPtr(s string) *onboard.StatusResponseInterestDetailsStatus {
	if s == "" {
		return nil
	}
	v := onboard.StatusResponseInterestDetailsStatus(s)
	return &v
}

func accountStatusPtr(s string) *onboard.StatusResponseAccountOnboardingStatus {
	if s == "" {
		return nil
	}
	v := onboard.StatusResponseAccountOnboardingStatus(s)
	return &v
}
