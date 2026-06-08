package customer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/org/card-onboarding-services/customer-management-service/pkg/customer"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler()
	customer.RegisterHandlers(r, h)
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

func TestRegisterCustomer_Success(t *testing.T) {
	r := setupTestRouter()

	payload := customer.RegisterCustomerRequest{
		CustomerId:    "CUST001",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
		CorrelationId: "corr-123",
	}
	bodyBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/internal/customers/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-Id", "corr-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp customer.RegisterCustomerResult
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if resp.CustomerId != "CUST001" {
		t.Errorf("Expected CustomerId CUST001, got %s", resp.CustomerId)
	}
	if resp.CoreCustomerId != "CORE-CUST001" {
		t.Errorf("Expected CoreCustomerId CORE-CUST001, got %s", resp.CoreCustomerId)
	}
	if resp.Status != "REGISTERED" {
		t.Errorf("Expected status REGISTERED, got %s", resp.Status)
	}
}

func TestRegisterCustomer_FailRegister(t *testing.T) {
	r := setupTestRouter()

	payload := customer.RegisterCustomerRequest{
		CustomerId:    "CUST_FAIL_REGISTER",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
		CorrelationId: "corr-123",
	}
	bodyBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/internal/customers/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var resp customer.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if resp.Code != "CUST_FAIL_REGISTER" {
		t.Errorf("Expected code CUST_FAIL_REGISTER, got %s", resp.Code)
	}
}

func TestRegisterCustomer_BadRequest(t *testing.T) {
	r := setupTestRouter()

	payload := customer.RegisterCustomerRequest{
		CustomerId:    "CUST_BAD_REQUEST",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
		CorrelationId: "corr-123",
	}
	bodyBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/internal/customers/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp customer.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if resp.Code != "CUST_BAD_REQUEST" {
		t.Errorf("Expected code CUST_BAD_REQUEST, got %s", resp.Code)
	}
}

func TestGetCustomer_Success(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/internal/customers/CUST001", nil)
	req.Header.Set("X-Correlation-Id", "corr-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp customer.RegisterCustomerResult
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if resp.CustomerId != "CUST001" {
		t.Errorf("Expected CustomerId CUST001, got %s", resp.CustomerId)
	}
}

func TestGetCustomer_Failures(t *testing.T) {
	r := setupTestRouter()

	tests := []struct {
		customerID string
		expectCode int
		expectErr  string
	}{
		{"CUST_FAIL_REGISTER", http.StatusInternalServerError, "CUST_FAIL_REGISTER"},
		{"CUST_BAD_REQUEST", http.StatusBadRequest, "CUST_BAD_REQUEST"},
		{"CUST_NOT_FOUND", http.StatusNotFound, "CUST_NOT_FOUND"},
	}

	for _, tc := range tests {
		t.Run(tc.customerID, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/internal/customers/"+tc.customerID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectCode {
				t.Errorf("Expected status %d, got %d", tc.expectCode, w.Code)
			}

			var resp customer.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			if resp.Code != tc.expectErr {
				t.Errorf("Expected error code %s, got %s", tc.expectErr, resp.Code)
			}
		})
	}
}
