package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/security"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (services *AppServices) ExportRecoveryKey(pin string) (string, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return "", mapError(err)
	}
	key, err := services.vault.ExportRecoveryKey(pin)
	if err != nil {
		return "", mapError(err)
	}
	return key, nil
}

func (services *AppServices) CreateBackupDialog(ctx context.Context, pin string) (security.BackupManifest, error) {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return security.BackupManifest{}, mapError(err)
	}
	if err := services.vault.Persist(); err != nil {
		services.mu.Unlock()
		return security.BackupManifest{}, mapError(err)
	}
	recoveryRaw, err := services.vault.ResolveRecoveryKey(pin, "")
	if err != nil {
		services.mu.Unlock()
		return security.BackupManifest{}, mapError(err)
	}
	unitCount := 0
	attachmentCount := 0
	if services.vaultDB != nil {
		units, listErr := services.vaultDB.ListUnits()
		if listErr == nil {
			unitCount += len(units)
		}
		deleted, deletedErr := services.vaultDB.ListDeletedUnits()
		if deletedErr == nil {
			unitCount += len(deleted)
		}
		attachments, attachErr := services.vaultDB.ListAllAttachments()
		if attachErr == nil {
			attachmentCount = len(attachments)
		}
	}
	source := security.BackupSource{
		MetaPath:        services.vault.MetaPath(),
		EncDBPath:       services.vault.EncDBPath(),
		AttachmentsDir:  services.vault.AttachmentsDir(),
		UnitCount:       unitCount,
		AttachmentCount: attachmentCount,
	}
	services.mu.Unlock()

	destination, dialogErr := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Save MetalTracker backup",
		DefaultFilename: "metaltracker-" + time.Now().UTC().Format("20060102") + ".mtbackup",
		Filters: []runtime.FileFilter{
			{DisplayName: "MetalTracker Backup", Pattern: "*.mtbackup"},
		},
	})
	if dialogErr != nil {
		return security.BackupManifest{}, mapError(dialogErr)
	}
	if strings.TrimSpace(destination) == "" {
		return security.BackupManifest{}, apperr.Cancelled()
	}
	if !strings.HasSuffix(strings.ToLower(destination), ".mtbackup") {
		destination += ".mtbackup"
	}

	manifest, createErr := security.CreateBackupFile(destination, recoveryRaw, source)
	if createErr != nil {
		return security.BackupManifest{}, apperr.Internal(createErr.Error())
	}
	return manifest, nil
}

func (services *AppServices) VerifyBackupDialog(ctx context.Context, recoveryKey string) (security.BackupVerifyResult, error) {
	selectedPath, dialogErr := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Verify MetalTracker backup",
		Filters: []runtime.FileFilter{
			{DisplayName: "MetalTracker Backup", Pattern: "*.mtbackup"},
		},
	})
	if dialogErr != nil {
		return security.BackupVerifyResult{}, mapError(dialogErr)
	}
	if strings.TrimSpace(selectedPath) == "" {
		return security.BackupVerifyResult{}, apperr.Cancelled()
	}
	recoveryRaw, err := security.ParseRecoveryKey(strings.TrimSpace(recoveryKey))
	if err != nil {
		return security.BackupVerifyResult{}, mapError(err)
	}
	result, verifyErr := security.VerifyBackupFile(selectedPath, recoveryRaw)
	if verifyErr != nil {
		return security.BackupVerifyResult{
			Valid:   false,
			Message: verifyErr.Error(),
		}, nil
	}
	return result, nil
}

func (services *AppServices) RestoreBackupDialog(ctx context.Context, recoveryKey string, confirmReplace bool) (security.BackupManifest, error) {
	if !confirmReplace {
		return security.BackupManifest{}, apperr.Validation("Confirm replace to restore the backup.")
	}
	selectedPath, dialogErr := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Restore MetalTracker backup",
		Filters: []runtime.FileFilter{
			{DisplayName: "MetalTracker Backup", Pattern: "*.mtbackup"},
		},
	})
	if dialogErr != nil {
		return security.BackupManifest{}, mapError(dialogErr)
	}
	if strings.TrimSpace(selectedPath) == "" {
		return security.BackupManifest{}, apperr.Cancelled()
	}
	recoveryRaw, err := security.ParseRecoveryKey(strings.TrimSpace(recoveryKey))
	if err != nil {
		return security.BackupManifest{}, mapError(err)
	}

	// Verify before tearing down the current vault.
	if _, verifyErr := security.VerifyBackupFile(selectedPath, recoveryRaw); verifyErr != nil {
		return security.BackupManifest{}, apperr.Validation(verifyErr.Error())
	}

	services.mu.Lock()
	defer services.mu.Unlock()

	services.stopAutoLockLocked()
	if services.vaultDB != nil {
		_ = services.vault.Persist()
		_ = services.vaultDB.Close()
		services.vaultDB = nil
	}
	_ = services.vault.Lock()

	manifest, restoreErr := security.RestoreBackupFile(selectedPath, recoveryRaw, services.dataDir)
	if restoreErr != nil {
		return security.BackupManifest{}, apperr.Internal(restoreErr.Error())
	}

	// Ensure session plaintext is gone after restore.
	_ = os.Remove(filepath.Join(services.dataDir, "vault.session.db"))
	return manifest, nil
}

func (services *AppServices) SaveRecoveryKitDialog(ctx context.Context, pin string) error {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return mapError(err)
	}
	recoveryKey, err := services.vault.ExportRecoveryKey(pin)
	services.mu.Unlock()
	if err != nil {
		return mapError(err)
	}

	destination, dialogErr := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Save recovery kit",
		DefaultFilename: "metaltracker-recovery-kit.html",
		Filters: []runtime.FileFilter{
			{DisplayName: "HTML", Pattern: "*.html"},
		},
	})
	if dialogErr != nil {
		return mapError(dialogErr)
	}
	if strings.TrimSpace(destination) == "" {
		return apperr.Cancelled()
	}
	if !strings.HasSuffix(strings.ToLower(destination), ".html") {
		destination += ".html"
	}
	html := security.BuildRecoveryKitHTML(recoveryKey, time.Now().UTC().Format(time.RFC3339))
	if writeErr := os.WriteFile(destination, []byte(html), 0o600); writeErr != nil {
		return apperr.Internal(writeErr.Error())
	}
	return nil
}

func (services *AppServices) SaveRecoveryKitFromKeyDialog(ctx context.Context, recoveryKey string) error {
	parsed, err := security.ParseRecoveryKey(strings.TrimSpace(recoveryKey))
	if err != nil {
		return mapError(err)
	}
	encoded := security.EncodeRecoveryKey(parsed)

	destination, dialogErr := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Save recovery kit",
		DefaultFilename: "metaltracker-recovery-kit.html",
		Filters: []runtime.FileFilter{
			{DisplayName: "HTML", Pattern: "*.html"},
		},
	})
	if dialogErr != nil {
		return mapError(dialogErr)
	}
	if strings.TrimSpace(destination) == "" {
		return apperr.Cancelled()
	}
	if !strings.HasSuffix(strings.ToLower(destination), ".html") {
		destination += ".html"
	}
	html := security.BuildRecoveryKitHTML(encoded, time.Now().UTC().Format(time.RFC3339))
	if writeErr := os.WriteFile(destination, []byte(html), 0o600); writeErr != nil {
		return apperr.Internal(writeErr.Error())
	}
	return nil
}
