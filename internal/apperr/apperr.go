package apperr

import (
	"errors"
	"fmt"
	"strings"
)

const (
	CodeInvalidPIN        = "invalid_pin"
	CodeVaultLocked       = "vault_locked"
	CodeVaultExists       = "vault_exists"
	CodeVaultMissing      = "vault_missing"
	CodeWeakPIN           = "weak_pin"
	CodeInvalidRecovery   = "invalid_recovery"
	CodePriceUnavailable  = "price_unavailable"
	CodePriceStale        = "price_stale"
	CodeInvalidAPIKey     = "invalid_api_key"
	CodeNotFound          = "not_found"
	CodeValidation        = "validation"
	CodeCancelled         = "cancelled"
	CodeInternal          = "internal"
	CodeUpdateUnavailable = "update_unavailable"
)

// Error is a stable, UI-parseable application error: "code: message".
type Error struct {
	Code    string
	Message string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Message == "" {
		return err.Code
	}
	return err.Code + ": " + err.Message
}

func New(code string, message string) *Error {
	return &Error{Code: code, Message: message}
}

func Newf(code string, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func Validation(message string) *Error {
	return New(CodeValidation, message)
}

func NotFound(message string) *Error {
	return New(CodeNotFound, message)
}

func Internal(message string) *Error {
	return New(CodeInternal, message)
}

// Cancelled is returned when the user dismisses a file/save dialog.
func Cancelled() *Error {
	return New(CodeCancelled, "cancelled")
}

func PriceUnavailable(message string) *Error {
	return New(CodePriceUnavailable, message)
}

// As returns the *Error if present in the chain.
func As(err error) (*Error, bool) {
	var appError *Error
	if errors.As(err, &appError) {
		return appError, true
	}
	return nil, false
}

// Parse extracts code and message from an error (including Wails string form).
func Parse(err error) (code string, message string) {
	if err == nil {
		return "", ""
	}
	if appError, ok := As(err); ok {
		return appError.Code, appError.Message
	}
	text := err.Error()
	if code, message, ok := Split(text); ok {
		return code, message
	}
	return CodeInternal, text
}

// Split parses "code: message" when code is a known token.
func Split(text string) (code string, message string, ok bool) {
	code, message, found := strings.Cut(strings.TrimSpace(text), ": ")
	if !found {
		return "", text, false
	}
	switch code {
	case CodeInvalidPIN, CodeVaultLocked, CodeVaultExists, CodeVaultMissing,
		CodeWeakPIN, CodeInvalidRecovery, CodePriceUnavailable, CodePriceStale,
		CodeInvalidAPIKey, CodeNotFound, CodeValidation, CodeCancelled, CodeInternal,
		CodeUpdateUnavailable:
		return code, message, true
	default:
		return "", text, false
	}
}
