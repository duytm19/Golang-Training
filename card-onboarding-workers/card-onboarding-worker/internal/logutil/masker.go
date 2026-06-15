package logutil

import "strings"

// MaskCardNumber masks all digits of a card number except the last 4 digits.
// If the card number is shorter than 4 digits, it masks the entire string.
func MaskCardNumber(cardNumber string) string {
	length := len(cardNumber)
	if length <= 4 {
		return strings.Repeat("*", length)
	}
	maskedLength := length - 4
	return strings.Repeat("*", maskedLength) + cardNumber[maskedLength:]
}
