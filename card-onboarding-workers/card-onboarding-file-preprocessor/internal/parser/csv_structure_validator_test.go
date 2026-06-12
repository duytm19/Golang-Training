package parser

import (
	"encoding/csv"
	"strings"
	"testing"
)

func TestValidateCSV_HappyPath(t *testing.T) {
	csvData := "customer_id,card_type,card_number,expiry_date,holder_name,email\n" +
		"CUST001,VISA,4111111111111111,12/28,Nguyen Van A,a@example.com\n" +
		"CUST002,MASTERCARD,5555555555555555,10/27,Nguyen Van B,b@example.com\n"

	records, resultCSV, err := ValidateCSV([]byte(csvData), "cards.csv", "job-123", 1000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}

	// Verify first record
	r1 := records[0]
	if r1.CustomerId != "CUST001" || r1.RecordId != "REC-job-123-0002" || r1.CardType != "VISA" {
		t.Errorf("Unexpected record 1: %+v", r1)
	}

	// Verify result CSV structure
	r := csv.NewReader(strings.NewReader(string(resultCSV)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse result CSV: %v", err)
	}

	if len(rows) != 3 { // Header + 2 rows
		t.Fatalf("Expected 3 rows in result CSV, got %d", len(rows))
	}

	if rows[1][3] != "ACCEPTED" || rows[2][3] != "ACCEPTED" {
		t.Errorf("Expected status ACCEPTED, got %s and %s", rows[1][3], rows[2][3])
	}
}

func TestValidateCSV_InvalidExtension(t *testing.T) {
	_, _, err := ValidateCSV([]byte("data"), "cards.txt", "job-123", 1000)
	if err == nil || !strings.Contains(err.Error(), "invalid file extension") {
		t.Errorf("Expected invalid extension error, got %v", err)
	}
}

func TestValidateCSV_EmptyFile(t *testing.T) {
	_, _, err := ValidateCSV([]byte(""), "cards.csv", "job-123", 1000)
	if err == nil || !strings.Contains(err.Error(), "empty file") {
		t.Errorf("Expected empty file error, got %v", err)
	}
}

func TestValidateCSV_FileTooLarge(t *testing.T) {
	_, _, err := ValidateCSV([]byte("customer_id,card_type"), "cards.csv", "job-123", 5)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected file size exceeds limit error, got %v", err)
	}
}

func TestValidateCSV_InvalidHeaderFormat(t *testing.T) {
	// Wrong columns count in header
	csvData := "customer_id,card_type,card_number\n"
	_, _, err := ValidateCSV([]byte(csvData), "cards.csv", "job-123", 1000)
	if err == nil || !strings.Contains(err.Error(), "invalid header format") {
		t.Errorf("Expected invalid header columns count error, got %v", err)
	}

	// Mismatched header name
	csvData = "cust_id,card_type,card_number,expiry_date,holder_name,email\n"
	_, _, err = ValidateCSV([]byte(csvData), "cards.csv", "job-123", 1000)
	if err == nil || !strings.Contains(err.Error(), "invalid header column") {
		t.Errorf("Expected header column name mismatch error, got %v", err)
	}
}

func TestValidateCSV_RowValidations(t *testing.T) {
	csvData := "customer_id,card_type,card_number,expiry_date,holder_name,email\n" +
		"CUST001,VISA,4111,12/28,Name1,email1@example.com\n" + // Accepted
		"CUST002,MASTERCARD,5555,10/27,Name2\n" +             // Rejected (wrong columns count)
		",AMEX,3333,08/26,Name3,email3@example.com\n"          // Rejected (missing customer_id)

	records, resultCSV, err := ValidateCSV([]byte(csvData), "cards.csv", "job-123", 1000)
	if err != nil {
		t.Fatalf("Expected no structural file error, got %v", err)
	}

	// Only 1 record should be accepted
	if len(records) != 1 {
		t.Errorf("Expected 1 accepted record, got %d", len(records))
	}
	if records[0].CustomerId != "CUST001" {
		t.Errorf("Expected first record to be CUST001, got %s", records[0].CustomerId)
	}

	// Result CSV should list 3 rows with their statuses
	r := csv.NewReader(strings.NewReader(string(resultCSV)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse result CSV: %v", err)
	}

	if len(rows) != 4 { // Header + 3 rows
		t.Fatalf("Expected 4 rows in result CSV, got %d", len(rows))
	}

	// Row 2 (CUST001) -> ACCEPTED
	if rows[1][3] != "ACCEPTED" {
		t.Errorf("Expected row 2 to be ACCEPTED, got %s", rows[1][3])
	}
	// Row 3 (CUST002) -> REJECTED (invalid column count)
	if rows[2][3] != "REJECTED" || !strings.Contains(rows[2][4], "invalid column count") {
		t.Errorf("Expected row 3 to be REJECTED with wrong count, got %s (%s)", rows[2][3], rows[2][4])
	}
	// Row 4 (empty customer_id) -> REJECTED (missing customer_id)
	if rows[3][3] != "REJECTED" || !strings.Contains(rows[3][4], "missing customer_id") {
		t.Errorf("Expected row 4 to be REJECTED with missing ID, got %s (%s)", rows[3][3], rows[3][4])
	}
}
