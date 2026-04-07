package validation

import (
	"unicode"
)

// IsLuhnValid проверяет номер заказа на соответствие алгоритму Луна
func IsLuhnValid(orderID string) bool {
	var sum int
	shouldDouble := false

	for i := len(orderID) - 1; i >= 0; i-- {
		r := rune(orderID[i])

		if !unicode.IsDigit(r) {
			continue
		}

		digit := int(r - '0')

		if shouldDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		shouldDouble = !shouldDouble
	}

	return sum%10 == 0
}
