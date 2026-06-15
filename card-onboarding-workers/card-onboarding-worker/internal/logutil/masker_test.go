package logutil

import "testing"

func TestMaskCardNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "16-digit card number",
			input:    "4111111111111111",
			expected: "************1111",
		},
		{
			name:     "13-digit card number",
			input:    "1234567890123",
			expected: "*********0123",
		},
		{
			name:     "19-digit card number",
			input:    "1234567890123456789",
			expected: "***************6789",
		},
		{
			name:     "Exactly 4 digits",
			input:    "1234",
			expected: "****",
		},
		{
			name:     "Less than 4 digits",
			input:    "12",
			expected: "**",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := MaskCardNumber(tt.input)
			if actual != tt.expected {
				t.Errorf("MaskCardNumber(%q) = %q, expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}
