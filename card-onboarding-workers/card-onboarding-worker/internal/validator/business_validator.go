package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
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

// ValidateCardRecord performs business validations on the card record.
func ValidateCardRecord(rec CardRecord, now time.Time) error {
	// 1. Card Type Validation
	switch rec.CardType {
	case "VISA", "MASTERCARD", "AMEX":
		// Valid
	default:
		return fmt.Errorf("invalid card type: %q (must be VISA, MASTERCARD, or AMEX)", rec.CardType)
	}

	// 2. Card Number Validation
	if !numericRegex.MatchString(rec.CardNumber) {
		return errors.New("card number must be numeric")
	}
	cardLen := len(rec.CardNumber)
	if cardLen < 13 || cardLen > 19 {
		return fmt.Errorf("invalid card number length: %d (must be between 13 and 19)", cardLen)
	}

	// 3. Expiry Date Validation (MM/YY)
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

	// Assume 21st century (2000s)
	expiryYear := 2000 + yearLastTwo
	expiryMonth := time.Month(month)

	// A card is valid until the last day of the expiry month.
	// To check expiration, we verify if the expiry month/year is before the current month/year.
	curYear := now.Year()
	curMonth := int(now.Month())

	if expiryYear < curYear || (expiryYear == curYear && int(expiryMonth) < curMonth) {
		return fmt.Errorf("card has expired: %s (current date: %s)", rec.ExpiryDate, now.Format("01/06"))
	}

	// 4. Email Validation
	if !emailRegex.MatchString(rec.Email) {
		return fmt.Errorf("invalid email format: %q", rec.Email)
	}

	return nil
}
