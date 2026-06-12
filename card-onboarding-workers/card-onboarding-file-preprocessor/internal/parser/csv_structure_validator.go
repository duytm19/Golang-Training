package parser

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
)

// ParsedRecord represents a structurally valid cardholder record parsed from CSV.
type ParsedRecord struct {
	RecordId   string `json:"recordId"`
	RowNumber  int    `json:"rowNumber"`
	CustomerId string `json:"customerId"`
	CardType   string `json:"cardType"`
	CardNumber string `json:"cardNumber"`
	ExpiryDate string `json:"expiryDate"`
	HolderName string `json:"holderName"`
	Email      string `json:"email"`
}

// PreprocessRowResult represents the preprocess outcome of a single CSV row.
type PreprocessRowResult struct {
	RecordId     string
	RowNumber    int
	CustomerId   string
	Status       string // ACCEPTED or REJECTED
	ErrorMessage string
}

// ValidateCSV validates file metadata, structure, headers, and splits accepted rows.
func ValidateCSV(data []byte, fileName string, jobId string, maxSizeBytes int64) ([]ParsedRecord, []byte, error) {
	// 1. File Extension check
	if !strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		return nil, nil, errors.New("invalid file extension: must be .csv")
	}

	// 2. File Size check
	fileSize := int64(len(data))
	if fileSize == 0 {
		return nil, nil, errors.New("invalid file size: empty file")
	}
	if fileSize > maxSizeBytes {
		return nil, nil, fmt.Errorf("invalid file size: exceeds limit of %d bytes", maxSizeBytes)
	}

	// 3. Parse CSV rows
	reader := csv.NewReader(bytes.NewReader(data))
	// LazyQuotes allows fields with unescaped double quotes inside them
	reader.LazyQuotes = true
	// FieldsPerRecord = -1 disables validation of field count by the reader
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CSV structure: %w", err)
	}

	// 4. Header Validation
	if len(records) == 0 {
		return nil, nil, errors.New("invalid CSV content: no rows found")
	}

	headers := records[0]
	expectedHeaders := []string{"customer_id", "card_type", "card_number", "expiry_date", "holder_name", "email"}
	if len(headers) != len(expectedHeaders) {
		return nil, nil, fmt.Errorf("invalid header format: expected %d columns, got %d", len(expectedHeaders), len(headers))
	}

	for i, h := range headers {
		if strings.TrimSpace(strings.ToLower(h)) != expectedHeaders[i] {
			return nil, nil, fmt.Errorf("invalid header column at index %d: expected %q, got %q", i, expectedHeaders[i], h)
		}
	}

	var parsedRecords []ParsedRecord
	var rowResults []PreprocessRowResult

	// 5. Process data rows (headers are row 1, data starts at row 2)
	for idx, row := range records[1:] {
		rowNumber := idx + 2 // 1-indexed row number
		customerId := ""
		recordId := fmt.Sprintf("REC-%s-%04d", jobId, rowNumber)

		if len(row) > 0 {
			customerId = strings.TrimSpace(row[0])
		}

		// Row column count check
		if len(row) != len(expectedHeaders) {
			rowResults = append(rowResults, PreprocessRowResult{
				RecordId:     recordId,
				RowNumber:    rowNumber,
				CustomerId:   customerId,
				Status:       "REJECTED",
				ErrorMessage: fmt.Sprintf("invalid column count: expected 6, got %d", len(row)),
			})
			continue
		}

		// Customer ID validation (structural requirement)
		if customerId == "" {
			rowResults = append(rowResults, PreprocessRowResult{
				RecordId:     recordId,
				RowNumber:    rowNumber,
				CustomerId:   customerId,
				Status:       "REJECTED",
				ErrorMessage: "missing customer_id",
			})
			continue
		}

		// If checks pass, row is accepted
		rowResults = append(rowResults, PreprocessRowResult{
			RecordId:     recordId,
			RowNumber:    rowNumber,
			CustomerId:   customerId,
			Status:       "ACCEPTED",
			ErrorMessage: "",
		})

		parsedRecords = append(parsedRecords, ParsedRecord{
			RecordId:   recordId,
			RowNumber:  rowNumber,
			CustomerId: customerId,
			CardType:   strings.TrimSpace(row[1]),
			CardNumber: strings.TrimSpace(row[2]),
			ExpiryDate: strings.TrimSpace(row[3]),
			HolderName: strings.TrimSpace(row[4]),
			Email:      strings.TrimSpace(row[5]),
		})
	}

	// 6. Generate result CSV file
	var resultBuf bytes.Buffer
	writer := csv.NewWriter(&resultBuf)

	// Write Header
	err = writer.Write([]string{"record_id", "row_number", "customer_id", "preprocess_status", "error_message"})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write result header: %w", err)
	}

	for _, rr := range rowResults {
		err = writer.Write([]string{
			rr.RecordId,
			fmt.Sprintf("%d", rr.RowNumber),
			rr.CustomerId,
			rr.Status,
			rr.ErrorMessage,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to write result row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, nil, fmt.Errorf("failed to finalize result writer: %w", err)
	}

	return parsedRecords, resultBuf.Bytes(), nil
}
