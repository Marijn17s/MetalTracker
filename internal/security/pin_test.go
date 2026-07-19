package security

import "testing"

func TestValidatePINExactlySixDigits(t *testing.T) {
	if err := ValidatePIN("123456"); err != nil {
		t.Fatalf("valid PIN rejected: %v", err)
	}
	for _, pin := range []string{"", "12345", "1234567", "12a456", "abcdef"} {
		if err := ValidatePIN(pin); err == nil {
			t.Fatalf("expected rejection for %q", pin)
		}
	}
}
