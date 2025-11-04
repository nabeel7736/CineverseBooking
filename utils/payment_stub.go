package utils

import (
	"fmt"
	"time"
)

// ProcessPaymentStub simulates payment success
func ProcessPaymentStub(amount float64, method, payload string) (string, error) {
	txRef := fmt.Sprintf("PAY-%d", time.Now().UnixNano())
	return txRef, nil
}
