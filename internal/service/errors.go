package service

import (
	"errors"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/security"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := apperr.As(err); ok {
		return err
	}
	switch {
	case errors.Is(err, security.ErrInvalidPIN):
		return apperr.New(apperr.CodeInvalidPIN, "Invalid PIN")
	case errors.Is(err, security.ErrInvalidRecovery):
		return apperr.New(apperr.CodeInvalidRecovery, "Invalid recovery key")
	case errors.Is(err, security.ErrVaultLocked):
		return apperr.New(apperr.CodeVaultLocked, "Vault is locked")
	case errors.Is(err, security.ErrVaultExists):
		return apperr.New(apperr.CodeVaultExists, "Vault already exists")
	case errors.Is(err, security.ErrVaultMissing):
		return apperr.New(apperr.CodeVaultMissing, "Vault does not exist")
	case errors.Is(err, security.ErrWeakPIN):
		return apperr.New(apperr.CodeWeakPIN, security.ErrWeakPIN.Error())
	default:
		return err
	}
}
