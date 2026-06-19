package util

import "strings"

// MaskCardNumber masks all digits of a card number except the last 4.
// A card number shorter than or equal to 4 digits is fully masked.
// Example: 4111111111111111 -> ************1111
func MaskCardNumber(cardNumber string) string {
	length := len(cardNumber)
	if length <= 4 {
		return strings.Repeat("*", length)
	}
	maskedLength := length - 4
	return strings.Repeat("*", maskedLength) + cardNumber[maskedLength:]
}
