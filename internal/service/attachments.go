package service

import (
	"context"
	"encoding/base64"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/domain"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (services *AppServices) ListAttachments(ownerType string, ownerID string) ([]domain.Attachment, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return nil, mapError(err)
	}
	if err := validateAttachmentOwner(ownerType, ownerID); err != nil {
		return nil, err
	}
	items, err := services.vaultDB.ListAttachments(ownerType, ownerID)
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

func (services *AppServices) AddAttachmentFromDialog(
	ctx context.Context,
	ownerType string,
	ownerID string,
	kind string,
) (domain.Attachment, error) {
	if err := validateAttachmentOwner(ownerType, ownerID); err != nil {
		return domain.Attachment{}, err
	}
	normalizedKind, filters, err := attachmentKindFilters(kind)
	if err != nil {
		return domain.Attachment{}, err
	}

	services.mu.Lock()
	if unlockErr := services.requireUnlockedLocked(); unlockErr != nil {
		services.mu.Unlock()
		return domain.Attachment{}, mapError(unlockErr)
	}
	if ownerErr := services.ensureAttachmentOwnerExistsLocked(ownerType, ownerID); ownerErr != nil {
		services.mu.Unlock()
		return domain.Attachment{}, ownerErr
	}
	services.mu.Unlock()

	selectedPath, dialogErr := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title:   "Select " + normalizedKind,
		Filters: filters,
	})
	if dialogErr != nil {
		return domain.Attachment{}, mapError(dialogErr)
	}
	if strings.TrimSpace(selectedPath) == "" {
		return domain.Attachment{}, apperr.Cancelled()
	}

	plaintext, readErr := os.ReadFile(selectedPath)
	if readErr != nil {
		return domain.Attachment{}, apperr.Internal("Could not read the selected file.")
	}
	if len(plaintext) == 0 {
		return domain.Attachment{}, apperr.Validation("Selected file is empty.")
	}
	if len(plaintext) > 25*1024*1024 {
		return domain.Attachment{}, apperr.Validation("File is larger than 25 MB.")
	}

	filename := filepath.Base(selectedPath)
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.Attachment{}, mapError(err)
	}
	if err := services.ensureAttachmentOwnerExistsLocked(ownerType, ownerID); err != nil {
		return domain.Attachment{}, err
	}

	encrypted, encryptErr := services.vault.EncryptPayload(plaintext)
	if encryptErr != nil {
		return domain.Attachment{}, mapError(encryptErr)
	}

	attachmentDir := services.vault.AttachmentsDir()
	if mkdirErr := os.MkdirAll(attachmentDir, 0o700); mkdirErr != nil {
		return domain.Attachment{}, mapError(mkdirErr)
	}

	created, createErr := services.vaultDB.CreateAttachment(domain.Attachment{
		OwnerType:   ownerType,
		OwnerID:     ownerID,
		Kind:        normalizedKind,
		Filename:    filename,
		ContentType: contentType,
	})
	if createErr != nil {
		return domain.Attachment{}, mapError(createErr)
	}

	absolutePath := filepath.Join(attachmentDir, created.RelativePath)
	if writeErr := os.WriteFile(absolutePath, encrypted, 0o600); writeErr != nil {
		_ = services.vaultDB.DeleteAttachment(created.ID)
		return domain.Attachment{}, mapError(writeErr)
	}
	if err := services.vault.Persist(); err != nil {
		return domain.Attachment{}, mapError(err)
	}
	return created, nil
}

func (services *AppServices) GetAttachmentBytes(attachmentID string) (domain.AttachmentBytes, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.AttachmentBytes{}, mapError(err)
	}
	item, err := services.vaultDB.GetAttachment(attachmentID)
	if err != nil {
		return domain.AttachmentBytes{}, apperr.NotFound("attachment not found")
	}
	encrypted, readErr := os.ReadFile(filepath.Join(services.vault.AttachmentsDir(), item.RelativePath))
	if readErr != nil {
		return domain.AttachmentBytes{}, apperr.NotFound("attachment file missing")
	}
	plaintext, decryptErr := services.vault.DecryptPayload(encrypted)
	if decryptErr != nil {
		return domain.AttachmentBytes{}, mapError(decryptErr)
	}
	return domain.AttachmentBytes{
		ID:          item.ID,
		Filename:    item.Filename,
		ContentType: item.ContentType,
		DataBase64:  base64.StdEncoding.EncodeToString(plaintext),
	}, nil
}

func (services *AppServices) DeleteAttachment(attachmentID string) error {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	item, err := services.vaultDB.GetAttachment(attachmentID)
	if err != nil {
		return apperr.NotFound("attachment not found")
	}
	_ = os.Remove(filepath.Join(services.vault.AttachmentsDir(), item.RelativePath))
	if err := services.vaultDB.DeleteAttachment(attachmentID); err != nil {
		return mapError(err)
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) SaveAttachmentDialog(ctx context.Context, attachmentID string) error {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return mapError(err)
	}
	item, err := services.vaultDB.GetAttachment(attachmentID)
	if err != nil {
		services.mu.Unlock()
		return apperr.NotFound("attachment not found")
	}
	encrypted, readErr := os.ReadFile(filepath.Join(services.vault.AttachmentsDir(), item.RelativePath))
	if readErr != nil {
		services.mu.Unlock()
		return apperr.NotFound("attachment file missing")
	}
	plaintext, decryptErr := services.vault.DecryptPayload(encrypted)
	services.mu.Unlock()
	if decryptErr != nil {
		return mapError(decryptErr)
	}

	destination, dialogErr := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Save attachment",
		DefaultFilename: item.Filename,
	})
	if dialogErr != nil {
		return mapError(dialogErr)
	}
	if strings.TrimSpace(destination) == "" {
		return apperr.Cancelled()
	}
	if writeErr := os.WriteFile(destination, plaintext, 0o600); writeErr != nil {
		return apperr.Internal("Could not save the file.")
	}
	return nil
}

func (services *AppServices) removeAttachmentsForOwnerLocked(ownerType string, ownerID string) error {
	items, err := services.vaultDB.DeleteAttachmentsForOwner(ownerType, ownerID)
	if err != nil {
		return err
	}
	for _, item := range items {
		_ = os.Remove(filepath.Join(services.vault.AttachmentsDir(), item.RelativePath))
	}
	return nil
}

func (services *AppServices) ensureAttachmentOwnerExistsLocked(ownerType string, ownerID string) error {
	switch ownerType {
	case domain.AttachmentOwnerUnit:
		if _, err := services.vaultDB.GetUnit(ownerID); err != nil {
			return apperr.NotFound("unit not found")
		}
		return nil
	case domain.AttachmentOwnerInvestment:
		units, err := services.vaultDB.ListUnits()
		if err != nil {
			return mapError(err)
		}
		for _, unit := range units {
			if unit.InvestmentID == ownerID {
				return nil
			}
		}
		return apperr.NotFound("investment not found")
	default:
		return apperr.Validation("unsupported attachment owner")
	}
}

func validateAttachmentOwner(ownerType string, ownerID string) error {
	if strings.TrimSpace(ownerID) == "" {
		return apperr.Validation("owner id is required")
	}
	switch ownerType {
	case domain.AttachmentOwnerUnit, domain.AttachmentOwnerInvestment:
		return nil
	default:
		return apperr.Validation("unsupported attachment owner")
	}
}

func attachmentKindFilters(kind string) (string, []runtime.FileFilter, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case domain.AttachmentKindPhoto:
		return domain.AttachmentKindPhoto, []runtime.FileFilter{{
			DisplayName: "Images",
			Pattern:     "*.jpg;*.jpeg;*.png;*.webp;*.gif",
		}}, nil
	case domain.AttachmentKindReceipt:
		return domain.AttachmentKindReceipt, []runtime.FileFilter{{
			DisplayName: "Receipts",
			Pattern:     "*.pdf;*.jpg;*.jpeg;*.png;*.webp",
		}}, nil
	default:
		return "", nil, apperr.Validation("unsupported attachment kind")
	}
}
