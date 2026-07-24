package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/domain"
	"MetalTracker/internal/update"
	"MetalTracker/internal/version"
)

func (services *AppServices) GetAppVersion() string {
	return version.Display()
}

func (services *AppServices) CheckForUpdates(ctx context.Context) (domain.UpdateCheckResult, error) {
	client := update.NewClient()
	result, pending, err := client.Check(ctx)
	if err != nil {
		if errors.Is(err, update.ErrNoAsset) {
			return domain.UpdateCheckResult{CurrentVersion: version.Display()}, apperr.New(apperr.CodeUpdateUnavailable, "No update package is available for this platform yet.")
		}
		return domain.UpdateCheckResult{CurrentVersion: version.Display()}, apperr.New(apperr.CodeUpdateUnavailable, "Could not check for updates. Check your network connection and try again.")
	}

	services.mu.Lock()
	if result.Available {
		services.pendingUpdate = pending
	} else {
		services.pendingUpdate = nil
	}
	services.mu.Unlock()

	return result, nil
}

func (services *AppServices) InstallUpdate(ctx context.Context, onProgress update.ProgressFunc) (string, error) {
	services.mu.Lock()
	pending := services.pendingUpdate
	services.mu.Unlock()
	if pending == nil {
		return "", apperr.Validation("check for updates first")
	}

	kind := pending.Kind
	if kind == "" {
		kind = update.KindBinary
	}

	client := update.NewClient()
	if err := client.Apply(ctx, pending, onProgress); err != nil {
		return "", apperr.New(apperr.CodeUpdateUnavailable, "Could not install the update. Try downloading the latest release from GitHub instead.")
	}

	services.mu.Lock()
	services.pendingUpdate = nil
	services.mu.Unlock()
	return kind, nil
}

// RelaunchCurrentExecutable starts a new process for the current binary.
func RelaunchCurrentExecutable() error {
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(executablePath)
	command.Stdout = nil
	command.Stderr = nil
	command.Stdin = nil
	if err := command.Start(); err != nil {
		return err
	}
	// Give the child a moment to start before the parent exits.
	time.Sleep(400 * time.Millisecond)
	return nil
}
