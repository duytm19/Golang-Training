package account

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/account-management-service/pkg/account"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler()
	account.RegisterHandlers(r, h)
	return r
}

func TestHealthCheck(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if body["status"] != "UP" {
		t.Errorf("Expected status UP, got %s", body["status"])
	}
}

func TestGetInterestDetails_Success(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/internal/accounts/CUST001/interest-details", nil)
	req.Header.Set("X-Correlation-Id", "corr-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp account.InterestDetailsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if resp.CustomerId != "CUST001" {
		t.Errorf("Expected CustomerId CUST001, got %s", resp.CustomerId)
	}
	if resp.ProductCode != "SAVINGS_BASIC" {
		t.Errorf("Expected ProductCode SAVINGS_BASIC, got %s", resp.ProductCode)
	}
	if resp.InterestRate != 4.5 {
		t.Errorf("Expected InterestRate 4.5, got %f", resp.InterestRate)
	}
	if resp.InterestType != account.VARIABLE {
		t.Errorf("Expected InterestType VARIABLE, got %v", resp.InterestType)
	}
	if resp.Currency != "AUD" {
		t.Errorf("Expected Currency AUD, got %s", resp.Currency)
	}
}

func TestGetInterestDetails_Failures(t *testing.T) {
	r := setupTestRouter()

	tests := []struct {
		customerID string
		expectCode int
		expectErr  string
	}{
		{"CUST_FAIL_INTEREST", http.StatusInternalServerError, "CUST_FAIL_INTEREST"},
		{"CUST_NO_INTEREST", http.StatusNotFound, "CUST_NO_INTEREST"},
	}

	for _, tc := range tests {
		t.Run(tc.customerID, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/internal/accounts/"+tc.customerID+"/interest-details", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectCode {
				t.Errorf("Expected status %d, got %d", tc.expectCode, w.Code)
			}

			var resp account.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			if resp.Code != tc.expectErr {
				t.Errorf("Expected error code %s, got %s", tc.expectErr, resp.Code)
			}
		})
	}
}
