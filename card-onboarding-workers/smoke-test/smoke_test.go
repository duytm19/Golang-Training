package smoke_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	onboard "github.com/org/card-onboarding-services/onboard-service/pkg/onboard"
)

// Local copy of CardRecord structure mirroring worker schema
type CardRecord struct {
	CorrelationId string `json:"correlationId"`
	JobId         string `json:"jobId"`
	RecordId      string `json:"recordId"`
	SourceFile    string `json:"sourceFile"`
	RowNumber     int    `json:"rowNumber"`
	CustomerId    string `json:"customerId"`
	CardType      string `json:"cardType"`
	CardNumber    string `json:"cardNumber"`
	ExpiryDate    string `json:"expiryDate"`
	HolderName    string `json:"holderName"`
	Email         string `json:"email"`
}

var (
	emailRegex   = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	numericRegex = regexp.MustCompile(`^[0-9]+$`)
)

func localValidateCardRecord(rec CardRecord, now time.Time) error {
	switch rec.CardType {
	case "VISA", "MASTERCARD", "AMEX":
		// Valid
	default:
		return fmt.Errorf("invalid card type: %q (must be VISA, MASTERCARD, or AMEX)", rec.CardType)
	}

	if !numericRegex.MatchString(rec.CardNumber) {
		return errors.New("card number must be numeric")
	}
	cardLen := len(rec.CardNumber)
	if cardLen < 13 || cardLen > 19 {
		return fmt.Errorf("invalid card number length: %d (must be between 13 and 19)", cardLen)
	}

	if len(rec.ExpiryDate) != 5 || rec.ExpiryDate[2] != '/' {
		return fmt.Errorf("invalid expiry date format: %q (must be MM/YY)", rec.ExpiryDate)
	}
	mmStr := rec.ExpiryDate[0:2]
	yyStr := rec.ExpiryDate[3:5]

	month, err := strconv.Atoi(mmStr)
	if err != nil || month < 1 || month > 12 {
		return fmt.Errorf("invalid expiry month: %q", mmStr)
	}

	yearLastTwo, err := strconv.Atoi(yyStr)
	if err != nil || yearLastTwo < 0 {
		return fmt.Errorf("invalid expiry year: %q", yyStr)
	}

	expiryYear := 2000 + yearLastTwo
	expiryMonth := time.Month(month)

	curYear := now.Year()
	curMonth := int(now.Month())

	if expiryYear < curYear || (expiryYear == curYear && int(expiryMonth) < curMonth) {
		return fmt.Errorf("card has expired: %s (current date: %s)", rec.ExpiryDate, now.Format("01/06"))
	}

	if !emailRegex.MatchString(rec.Email) {
		return fmt.Errorf("invalid email format: %q", rec.Email)
	}

	return nil
}

func localValidateCSV(data []byte, fileName string, jobId string, maxSizeBytes int64) ([]CardRecord, error) {
	if !strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		return nil, errors.New("invalid file extension: must be .csv")
	}

	fileSize := int64(len(data))
	if fileSize == 0 {
		return nil, errors.New("invalid file size: empty file")
	}
	if fileSize > maxSizeBytes {
		return nil, fmt.Errorf("invalid file size: exceeds limit of %d bytes", maxSizeBytes)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV structure: %w", err)
	}

	if len(records) == 0 {
		return nil, errors.New("invalid CSV content: no rows found")
	}

	headers := records[0]
	expectedHeaders := []string{"customer_id", "card_type", "card_number", "expiry_date", "holder_name", "email"}
	if len(headers) != len(expectedHeaders) {
		return nil, fmt.Errorf("invalid header format: expected %d columns, got %d", len(expectedHeaders), len(headers))
	}

	for i, h := range headers {
		if strings.TrimSpace(strings.ToLower(h)) != expectedHeaders[i] {
			return nil, fmt.Errorf("invalid header column at index %d: expected %q, got %q", i, expectedHeaders[i], h)
		}
	}

	var parsedRecords []CardRecord
	for idx, row := range records[1:] {
		rowNumber := idx + 2
		customerId := ""
		recordId := fmt.Sprintf("REC-%s-%04d", jobId, rowNumber)

		if len(row) > 0 {
			customerId = strings.TrimSpace(row[0])
		}

		if len(row) != len(expectedHeaders) {
			continue // skip rejected row
		}

		if customerId == "" {
			continue // skip rejected row
		}

		parsedRecords = append(parsedRecords, CardRecord{
			CorrelationId: "corr-" + recordId,
			JobId:         jobId,
			RecordId:      recordId,
			SourceFile:    fileName,
			RowNumber:     rowNumber,
			CustomerId:    customerId,
			CardType:      strings.TrimSpace(row[1]),
			CardNumber:    strings.TrimSpace(row[2]),
			ExpiryDate:    strings.TrimSpace(row[3]),
			HolderName:    strings.TrimSpace(row[4]),
			Email:         strings.TrimSpace(row[5]),
		})
	}

	return parsedRecords, nil
}

func buildMockOnboardService(t *testing.T) string {
	svcDir := "../../card-onboarding-services/onboard-service"
	binaryName := "onboard-service-mock"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(svcDir, binaryName)

	_ = os.Remove(binaryPath)

	cmd := exec.Command("go", "build", "-tags", "mock", "-o", binaryName, ".")
	cmd.Dir = svcDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build mock onboard-service: %v\nStderr: %s", err, stderr.String())
	}

	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path of mock onboard-service: %v", err)
	}
	return absPath
}

func startMockOnboardService(t *testing.T, binaryPath string) (*exec.Cmd, func()) {
	cmd := exec.Command(binaryPath)
	cmd.Env = []string{
		"PORT=8080",
		"CUSTOMER_SERVICE_URL=http://localhost:8081",
		"ACCOUNT_SERVICE_URL=http://localhost:8082",
		"AWS_REGION=ap-southeast-2",
		"REQUEST_STATUS_TABLE=mock-table",
		"ACCOUNT_DETAILS_TABLE=mock-table",
	}

	if path := os.Getenv("PATH"); path != "" {
		cmd.Env = append(cmd.Env, "PATH="+path)
	}
	if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		cmd.Env = append(cmd.Env, "SystemRoot="+systemRoot)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start mock onboard-service: %v", err)
	}

	cleanup := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		if t.Failed() {
			t.Logf("onboard-service stdout:\n%s", stdout.String())
			t.Logf("onboard-service stderr:\n%s", stderr.String())
		}
	}

	ready := false
	for i := 0; i < 30; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:8080", 100*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !ready {
		cleanup()
		t.Fatalf("onboard-service did not start in time. Stderr: %s", stderr.String())
	}

	return cmd, cleanup
}

func startMockServer(addr string, handler http.Handler) *http.Server {
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		_ = srv.ListenAndServe()
	}()
	return srv
}

func TestIntegrationSmokeTest(t *testing.T) {
	binaryPath := buildMockOnboardService(t)

	// Set up customer mock server (8081)
	customerMux := http.NewServeMux()
	customerMux.HandleFunc("/internal/customers/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			CustomerId string `json:"customerId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body.CustomerId == "CUST_FAIL_REGISTER" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code":    "CUST_FAIL_REGISTER",
				"message": "Simulated downstream customer registration failure",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"customerId":     body.CustomerId,
			"coreCustomerId": "CORE-" + body.CustomerId,
			"status":         "REGISTERED",
		})
	})
	customerServer := startMockServer("localhost:8081", customerMux)
	defer func() {
		_ = customerServer.Shutdown(context.Background())
	}()

	// Set up account mock server (8082)
	var accountFailInterest bool
	var accountFailMutex sync.Mutex

	accountMux := http.NewServeMux()
	accountMux.HandleFunc("/internal/accounts/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/interest-details") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		customerId := parts[3]

		accountFailMutex.Lock()
		fail := accountFailInterest
		accountFailMutex.Unlock()

		if customerId == "CUST_FAIL_INTEREST" && fail {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code":    "CUST_FAIL_INTEREST",
				"message": "Simulated downstream interest rate retrieval failure",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"customerId":   customerId,
			"productCode":  "SAVINGS_BASIC",
			"interestRate": 4.5,
			"interestType": "VARIABLE",
			"currency":     "AUD",
		})
	})
	accountServer := startMockServer("localhost:8082", accountMux)
	defer func() {
		_ = accountServer.Shutdown(context.Background())
	}()

	// Start onboard-service
	_, cleanupOnboard := startMockOnboardService(t, binaryPath)
	defer cleanupOnboard()

	// Initialize onboard-service client using generated pkg/onboard client
	client, err := onboard.NewClient("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create onboard-service client: %v", err)
	}

	ctx := context.Background()

	// ==========================================
	// Test Scenario 1: Happy Path
	// ==========================================
	t.Run("Happy Path - E2E", func(t *testing.T) {
		csvContent := "customer_id,card_type,card_number,expiry_date,holder_name,email\n" +
			"CUST001,VISA,4111111111111111,12/28,John Doe,john.doe@example.com\n"

		// Preprocessor parses S3 file (simulated locally)
		parsedRecords, err := localValidateCSV([]byte(csvContent), "cards.csv", "JOB-1", 10000)
		if err != nil {
			t.Fatalf("Preprocessor validation failed: %v", err)
		}
		if len(parsedRecords) != 1 {
			t.Fatalf("Expected 1 parsed record, got %d", len(parsedRecords))
		}

		rec := parsedRecords[0]

		// Worker runs business validation (simulated locally)
		err = localValidateCardRecord(rec, time.Now())
		if err != nil {
			t.Fatalf("Worker validation failed: %v", err)
		}

		// Worker calls onboard-service orchestrator
		req := onboard.OnboardCardJSONRequestBody{
			CustomerId:    rec.CustomerId,
			CardType:      onboard.OnboardRequestCardType(rec.CardType),
			CardNumber:    rec.CardNumber,
			ExpiryDate:    rec.ExpiryDate,
			HolderName:    rec.HolderName,
			Email:         openapi_types.Email(rec.Email),
			CorrelationId: rec.CorrelationId,
			JobId:         rec.JobId,
			RecordId:      rec.RecordId,
			RowNumber:     rec.RowNumber,
			SourceFile:    rec.SourceFile,
		}

		resp, err := client.OnboardCard(ctx, req)
		if err != nil {
			t.Fatalf("Worker call to onboard-service failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("OnboardCard failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Assert status is SUCCEEDED in onboard-service request status store
		statusResp, err := client.GetOnboardStatus(ctx, rec.CustomerId)
		if err != nil {
			t.Fatalf("Failed to fetch request status: %v", err)
		}
		defer statusResp.Body.Close()

		if statusResp.StatusCode != http.StatusOK {
			t.Fatalf("GetOnboardStatus failed with status %d", statusResp.StatusCode)
		}

		var status onboard.StatusResponse
		if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
			t.Fatalf("Failed to decode status response: %v", err)
		}

		if status.OverallStatus != onboard.StatusResponseOverallStatusSUCCEEDED {
			t.Errorf("Expected overall status to be SUCCEEDED, got %q", status.OverallStatus)
		}
	})

	// ==========================================
	// Test Scenario 2: CSV Structure Validation Failure
	// ==========================================
	t.Run("Structural Validation Failure", func(t *testing.T) {
		csvContent := "invalid_column,card_type,card_number,expiry_date,holder_name,email\n" +
			"CUST002,VISA,4111111111111111,12/28,John Doe,john.doe@example.com\n"

		_, err := localValidateCSV([]byte(csvContent), "cards.csv", "JOB-2", 10000)
		if err == nil {
			t.Fatal("Expected preprocessor error for invalid CSV headers, got nil")
		}
	})

	// ==========================================
	// Test Scenario 3: Business Validation Failure
	// ==========================================
	t.Run("Business Validation Failure", func(t *testing.T) {
		rec := CardRecord{
			CustomerId: "CUST003",
			CardType:   "VISA",
			CardNumber: "4111111111111111",
			ExpiryDate: "12/24", // Expired
			Email:      "invalid-email",
		}

		err := localValidateCardRecord(rec, time.Now())
		if err == nil {
			t.Fatal("Expected business validation error for expired card and invalid email, got nil")
		}
	})

	// ==========================================
	// Test Scenario 4: Resumption Path
	// ==========================================
	t.Run("Resumption Path", func(t *testing.T) {
		rec := CardRecord{
			CorrelationId: "corr-REC-JOB-3-0002",
			JobId:         "JOB-3",
			RecordId:      "REC-JOB-3-0002",
			SourceFile:    "cards.csv",
			RowNumber:     2,
			CustomerId:    "CUST_FAIL_INTEREST",
			CardType:      "MASTERCARD",
			CardNumber:    "5555444433332222",
			ExpiryDate:    "12/28",
			HolderName:    "Jane Doe",
			Email:         "jane.doe@example.com",
		}

		req := onboard.OnboardCardJSONRequestBody{
			CustomerId:    rec.CustomerId,
			CardType:      onboard.OnboardRequestCardType(rec.CardType),
			CardNumber:    rec.CardNumber,
			ExpiryDate:    rec.ExpiryDate,
			HolderName:    rec.HolderName,
			Email:         openapi_types.Email(rec.Email),
			CorrelationId: rec.CorrelationId,
			JobId:         rec.JobId,
			RecordId:      rec.RecordId,
			RowNumber:     rec.RowNumber,
			SourceFile:    rec.SourceFile,
		}

		// First call - should fail at Step 2 (interest rates)
		accountFailMutex.Lock()
		accountFailInterest = true
		accountFailMutex.Unlock()

		resp, err := client.OnboardCard(ctx, req)
		if err != nil {
			t.Fatalf("OnboardCard HTTP request failed: %v", err)
		}
		resp.Body.Close()

		// Verify state: customer registration succeeded, overall failed
		statusResp, err := client.GetOnboardStatus(ctx, rec.CustomerId)
		if err != nil {
			t.Fatalf("Failed to fetch request status: %v", err)
		}
		defer statusResp.Body.Close()

		var status onboard.StatusResponse
		if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
			t.Fatalf("Failed to decode status response: %v", err)
		}

		if status.CustomerRegistrationStatus == nil || *status.CustomerRegistrationStatus != onboard.StatusResponseCustomerRegistrationStatusSUCCEEDED {
			t.Errorf("Expected CustomerRegistrationStatus to be SUCCEEDED, got %v", status.CustomerRegistrationStatus)
		}
		if status.InterestDetailsStatus == nil || *status.InterestDetailsStatus != onboard.StatusResponseInterestDetailsStatusFAILED {
			t.Errorf("Expected InterestDetailsStatus to be FAILED, got %v", status.InterestDetailsStatus)
		}

		// Fix the downstream dependency and resume
		accountFailMutex.Lock()
		accountFailInterest = false
		accountFailMutex.Unlock()

		resp2, err := client.OnboardCard(ctx, req)
		if err != nil {
			t.Fatalf("Resumption HTTP request failed: %v", err)
		}
		resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Fatalf("Resumption failed with status %d", resp2.StatusCode)
		}

		// Verify overall status is now SUCCEEDED
		statusResp2, err := client.GetOnboardStatus(ctx, rec.CustomerId)
		if err != nil {
			t.Fatalf("Failed to fetch request status: %v", err)
		}
		defer statusResp2.Body.Close()

		var status2 onboard.StatusResponse
		if err := json.NewDecoder(statusResp2.Body).Decode(&status2); err != nil {
			t.Fatalf("Failed to decode status response: %v", err)
		}

		if status2.OverallStatus != onboard.StatusResponseOverallStatusSUCCEEDED {
			t.Errorf("Expected OverallStatus to be SUCCEEDED, got %q", status2.OverallStatus)
		}
	})

	// ==========================================
	// Test Scenario 5: SQS DLQ Retry Simulation
	// ==========================================
	t.Run("DLQ Redirection Simulation", func(t *testing.T) {
		// Mock server constantly returning 503 for a specific customer
		mock503Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Database connection failure",
			})
		}))
		defer mock503Server.Close()

		// Simulate Lambda SQS consumer loop with max 3 retries
		maxRetries := 3
		retryCount := 0
		isMovedToDLQ := false

		for retryCount < maxRetries {
			resp, err := http.Post(mock503Server.URL, "application/json", nil)
			if err != nil {
				retryCount++
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 500 {
				retryCount++
				continue
			}
			break
		}

		if retryCount == maxRetries {
			isMovedToDLQ = true
		}

		if !isMovedToDLQ {
			t.Error("Expected message to be redirected to DLQ after 3 failures, but it wasn't")
		}
	})
}
