package validator

import (
	"testing"
	"time"
)

func TestValidateCardRecord(t *testing.T) {
	// Base mock current time: June 15, 2026
	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		record    CardRecord
		expectErr bool
	}{
		{
			name: "Happy Path - VISA",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "12/28",
				Email:      "john.doe@example.com",
			},
			expectErr: false,
		},
		{
			name: "Happy Path - MASTERCARD",
			record: CardRecord{
				CardType:   "MASTERCARD",
				CardNumber: "5555444433332222",
				ExpiryDate: "06/26", // expires end of current month, should be valid
				Email:      "jane@domain.co",
			},
			expectErr: false,
		},
		{
			name: "Happy Path - AMEX",
			record: CardRecord{
				CardType:   "AMEX",
				CardNumber: "378282246310005",
				ExpiryDate: "01/30",
				Email:      "amex.user@test.org",
			},
			expectErr: false,
		},
		{
			name: "Invalid Card Type",
			record: CardRecord{
				CardType:   "DISCOVER",
				CardNumber: "6011111111111111",
				ExpiryDate: "12/28",
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Non-numeric Card Number",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "411111111111111a",
				ExpiryDate: "12/28",
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Card Number Too Short",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "123456789012", // 12 digits
				ExpiryDate: "12/28",
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Card Number Too Long",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "12345678901234567890", // 20 digits
				ExpiryDate: "12/28",
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Invalid Expiry Format - No Slash",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "12288",
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Invalid Expiry Format - Wrong Slash Position",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "1/228",
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Invalid Expiry Month",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "13/28",
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Card Expired in Past Year",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "12/25", // expired Dec 2025
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Card Expired in Previous Month",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "05/26", // expired May 2026
				Email:      "test@example.com",
			},
			expectErr: true,
		},
		{
			name: "Invalid Email - Missing Domain",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "12/28",
				Email:      "john.doe@",
			},
			expectErr: true,
		},
		{
			name: "Invalid Email - No At Symbol",
			record: CardRecord{
				CardType:   "VISA",
				CardNumber: "4111111111111111",
				ExpiryDate: "12/28",
				Email:      "john.doe.domain.com",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCardRecord(tt.record, now)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateCardRecord() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}
