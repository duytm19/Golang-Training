package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	emailRegex   = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	numericRegex = regexp.MustCompile(`^[0-9]+$`)
)

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

// ValidateCardRecord performs business validation on a card record per the
// documented rules (section 8.3): required fields plus card type, numeric card
// number, MM/YY expiry format, and email format.
func ValidateCardRecord(rec CardRecord) error {
	// 1. customerId is required
	if strings.TrimSpace(rec.CustomerId) == "" {
		return errors.New("customerId is required")
	}

	// 2. cardType is required and must be VISA, MASTERCARD, or AMEX
	switch rec.CardType {
	case "VISA", "MASTERCARD", "AMEX":
		// valid
	case "":
		return errors.New("cardType is required")
	default:
		return fmt.Errorf("invalid card type: %q (must be VISA, MASTERCARD, or AMEX)", rec.CardType)
	}

	// 3. cardNumber is required and must be numeric
	if rec.CardNumber == "" {
		return errors.New("cardNumber is required")
	}
	if !numericRegex.MatchString(rec.CardNumber) {
		return errors.New("cardNumber must be numeric")
	}

	// 4. expiryDate is required and must be MM/YY
	if rec.ExpiryDate == "" {
		return errors.New("expiryDate is required")
	}
	if len(rec.ExpiryDate) != 5 || rec.ExpiryDate[2] != '/' {
		return fmt.Errorf("invalid expiry date format: %q (must be MM/YY)", rec.ExpiryDate)
	}
	month, err := strconv.Atoi(rec.ExpiryDate[0:2])
	if err != nil || month < 1 || month > 12 {
		return fmt.Errorf("invalid expiry month: %q (must be MM/YY)", rec.ExpiryDate[0:2])
	}
	if _, err := strconv.Atoi(rec.ExpiryDate[3:5]); err != nil {
		return fmt.Errorf("invalid expiry year: %q (must be MM/YY)", rec.ExpiryDate[3:5])
	}

	// 5. holderName is required
	if strings.TrimSpace(rec.HolderName) == "" {
		return errors.New("holderName is required")
	}

	// 6. email is required and must be a valid email format
	if rec.Email == "" {
		return errors.New("email is required")
	}
	if !emailRegex.MatchString(rec.Email) {
		return fmt.Errorf("invalid email format: %q", rec.Email)
	}

	return nil
}
