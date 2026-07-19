package apperr

import (
	"errors"
	"testing"
)

func TestErrorFormatAndParse(t *testing.T) {
	err := Validation("quantity must be positive")
	if err.Error() != "validation: quantity must be positive" {
		t.Fatalf("unexpected error text: %s", err.Error())
	}

	code, message := Parse(err)
	if code != CodeValidation || message != "quantity must be positive" {
		t.Fatalf("parse mismatch: %s %s", code, message)
	}

	wrapped := errors.New("validation: bad input")
	code, message = Parse(wrapped)
	if code != CodeValidation || message != "bad input" {
		t.Fatalf("string parse mismatch: %s %s", code, message)
	}

	cancelled := Cancelled()
	code, message = Parse(cancelled)
	if code != CodeCancelled || message != "cancelled" {
		t.Fatalf("cancelled parse mismatch: %s %s", code, message)
	}
}
