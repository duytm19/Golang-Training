package validator

import (
	"testing"
)

func TestValidateCardRecord(t *testing.T) {
	base := CardRecord{
		CustomerId: "CUST001",
		CardType:   "VISA",
		CardNumber: "4111111111111111",
		ExpiryDate: "12/28",
		HolderName: "John Doe",
		Email:      "john.doe@example.com",
	}

	// withField returns a copy of base with one field overridden.
	withField := func(mutate func(r *CardRecord)) CardRecord {
		r := base
		mutate(&r)
		return r
	}

	tests := []struct {
		name      string
		record    CardRecord
		expectErr bool
	}{
		{
			name:      "Happy Path - VISA",
			record:    base,
			expectErr: false,
		},
		{
			name: "Happy Path - MASTERCARD",
			record: withField(func(r *CardRecord) {
				r.CardType = "MASTERCARD"
				r.CardNumber = "5555444433332222"
				r.Email = "jane@domain.co"
			}),
			expectErr: false,
		},
		{
			name: "Happy Path - AMEX",
			record: withField(func(r *CardRecord) {
				r.CardType = "AMEX"
				r.CardNumber = "378282246310005"
				r.ExpiryDate = "01/30"
				r.Email = "amex.user@test.org"
			}),
			expectErr: false,
		},
		{
			name:      "Missing customerId",
			record:    withField(func(r *CardRecord) { r.CustomerId = "" }),
			expectErr: true,
		},
		{
			name:      "Missing holderName",
			record:    withField(func(r *CardRecord) { r.HolderName = "" }),
			expectErr: true,
		},
		{
			name:      "Missing cardType",
			record:    withField(func(r *CardRecord) { r.CardType = "" }),
			expectErr: true,
		},
		{
			name:      "Invalid Card Type",
			record:    withField(func(r *CardRecord) { r.CardType = "DISCOVER" }),
			expectErr: true,
		},
		{
			name:      "Missing cardNumber",
			record:    withField(func(r *CardRecord) { r.CardNumber = "" }),
			expectErr: true,
		},
		{
			name:      "Non-numeric Card Number",
			record:    withField(func(r *CardRecord) { r.CardNumber = "411111111111111a" }),
			expectErr: true,
		},
		{
			name:      "Missing expiryDate",
			record:    withField(func(r *CardRecord) { r.ExpiryDate = "" }),
			expectErr: true,
		},
		{
			name:      "Invalid Expiry Format - No Slash",
			record:    withField(func(r *CardRecord) { r.ExpiryDate = "12288" }),
			expectErr: true,
		},
		{
			name:      "Invalid Expiry Format - Wrong Slash Position",
			record:    withField(func(r *CardRecord) { r.ExpiryDate = "1/228" }),
			expectErr: true,
		},
		{
			name:      "Invalid Expiry Month",
			record:    withField(func(r *CardRecord) { r.ExpiryDate = "13/28" }),
			expectErr: true,
		},
		{
			name:      "Invalid Email - Missing Domain",
			record:    withField(func(r *CardRecord) { r.Email = "john.doe@" }),
			expectErr: true,
		},
		{
			name:      "Invalid Email - No At Symbol",
			record:    withField(func(r *CardRecord) { r.Email = "john.doe.domain.com" }),
			expectErr: true,
		},
		{
			name:      "Missing email",
			record:    withField(func(r *CardRecord) { r.Email = "" }),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCardRecord(tt.record)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateCardRecord() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}
