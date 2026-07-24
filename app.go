package main

import (
	"context"

	"MetalTracker/internal/domain"
	"MetalTracker/internal/security"
	"MetalTracker/internal/service"
	"MetalTracker/internal/update"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	services *service.AppServices
}

func NewApp() *App {
	return &App{}
}

func (application *App) startup(ctx context.Context) {
	application.ctx = ctx
	services, err := service.NewAppServices()
	if err != nil {
		panic(err)
	}
	application.services = services
}

func (application *App) shutdown(ctx context.Context) {
	if application.services != nil {
		application.services.Close()
	}
}

func (application *App) VaultExists() bool {
	return application.services.VaultExists()
}

func (application *App) IsUnlocked() bool {
	return application.services.IsUnlocked()
}

func (application *App) SetupVault(pin string) (*security.SetupResult, error) {
	return application.services.SetupVault(pin)
}

func (application *App) Unlock(pin string) error {
	return application.services.Unlock(pin)
}

func (application *App) Lock() error {
	return application.services.Lock()
}

func (application *App) RecoverWithKey(recoveryKey string, newPIN string) error {
	return application.services.RecoverWithKey(recoveryKey, newPIN)
}

func (application *App) ChangePIN(currentPIN string, newPIN string) error {
	return application.services.ChangePIN(currentPIN, newPIN)
}

func (application *App) TouchActivity() {
	application.services.TouchActivity()
}

func (application *App) GetSettings() (domain.AppSettings, error) {
	return application.services.GetSettings()
}

func (application *App) UpdateSettings(settings domain.AppSettings) error {
	return application.services.UpdateSettings(settings)
}

func (application *App) GetLatestPrices() (domain.SpotQuote, error) {
	return application.services.GetLatestPrices(application.ctx)
}

func (application *App) CreateInvestment(request domain.CreateInvestmentRequest) (string, error) {
	return application.services.CreateInvestment(application.ctx, request)
}

func (application *App) ListGroupedHoldings(filter domain.HoldingsFilter) ([]domain.GroupedHolding, error) {
	return application.services.ListGroupedHoldings(application.ctx, filter)
}

func (application *App) GetHoldingsFilterOptions() (domain.HoldingsFilterOptions, error) {
	return application.services.GetHoldingsFilterOptions()
}

func (application *App) ListUnitsInGroup(productKey string) ([]domain.UnitValuation, error) {
	return application.services.ListUnitsInGroup(application.ctx, productKey)
}

func (application *App) GetUnit(unitID string) (domain.UnitValuation, error) {
	return application.services.GetUnit(application.ctx, unitID)
}

func (application *App) SellUnit(request domain.SellUnitRequest) error {
	return application.services.SellUnit(request)
}

func (application *App) SellUnits(request domain.SellUnitsRequest) error {
	return application.services.SellUnits(request)
}

func (application *App) BulkUpdateHoldingUnits(request domain.BulkUpdateHoldingUnitsRequest) error {
	return application.services.BulkUpdateHoldingUnits(request)
}

func (application *App) UpdateHoldingUnit(request domain.UpdateHoldingUnitRequest) error {
	return application.services.UpdateHoldingUnit(request)
}

func (application *App) DeleteHoldingUnit(unitID string) error {
	return application.services.DeleteHoldingUnit(unitID)
}

func (application *App) RestoreHoldingUnit(unitID string) error {
	return application.services.RestoreHoldingUnit(unitID)
}

func (application *App) PurgeHoldingUnit(unitID string) error {
	return application.services.PurgeHoldingUnit(unitID)
}

func (application *App) ListRecentlyDeleted() ([]domain.UnitValuation, error) {
	return application.services.ListRecentlyDeleted()
}

func (application *App) ListSoldArchive(filter domain.HoldingsFilter) ([]domain.UnitValuation, error) {
	return application.services.ListSoldArchive(application.ctx, filter)
}

func (application *App) ListAttachments(ownerType string, ownerID string) ([]domain.Attachment, error) {
	return application.services.ListAttachments(ownerType, ownerID)
}

func (application *App) AddAttachment(ownerType string, ownerID string, kind string) (domain.Attachment, error) {
	return application.services.AddAttachmentFromDialog(application.ctx, ownerType, ownerID, kind)
}

func (application *App) GetAttachmentBytes(attachmentID string) (domain.AttachmentBytes, error) {
	return application.services.GetAttachmentBytes(attachmentID)
}

func (application *App) DeleteAttachment(attachmentID string) error {
	return application.services.DeleteAttachment(attachmentID)
}

func (application *App) SaveAttachment(attachmentID string) error {
	return application.services.SaveAttachmentDialog(application.ctx, attachmentID)
}

func (application *App) ListDealerSummaries() ([]domain.DealerSummary, error) {
	return application.services.ListDealerSummaries(application.ctx)
}

func (application *App) GetPortfolioSummary() (domain.PortfolioSummary, error) {
	return application.services.GetPortfolioSummary(application.ctx)
}

func (application *App) GetPortfolioHistory(fromDate string, toDate string) ([]domain.PortfolioHistoryPoint, error) {
	return application.services.GetPortfolioHistory(application.ctx, fromDate, toDate)
}

func (application *App) GetPortfolioValueAt(date string) (domain.PortfolioValueAt, error) {
	return application.services.GetPortfolioValueAt(application.ctx, date)
}

func (application *App) GetMonthlyBreakdown(fromDate string, toDate string) ([]domain.MonthlyMetalBreakdown, error) {
	return application.services.GetMonthlyBreakdown(application.ctx, fromDate, toDate)
}

func (application *App) GetMonthlyPage(fromDate string, toDate string) (domain.MonthlyPage, error) {
	return application.services.GetMonthlyPage(application.ctx, fromDate, toDate)
}

func (application *App) GetAllocationBreakdown() (domain.AllocationBreakdown, error) {
	return application.services.GetAllocationBreakdown(application.ctx)
}

func (application *App) GetMetalAverageCosts() ([]domain.MetalAverageCost, error) {
	return application.services.GetMetalAverageCosts(application.ctx)
}

func (application *App) PreviewWhatIf(request domain.WhatIfRequest) (domain.WhatIfPreview, error) {
	return application.services.PreviewWhatIf(application.ctx, request)
}

func (application *App) GetPnLContribution(fromDate string, toDate string) (domain.PnLContributionReport, error) {
	return application.services.GetPnLContribution(application.ctx, fromDate, toDate)
}

func (application *App) ExportRecoveryKey(pin string) (string, error) {
	return application.services.ExportRecoveryKey(pin)
}

func (application *App) CreateBackup(pin string) (security.BackupManifest, error) {
	return application.services.CreateBackupDialog(application.ctx, pin)
}

func (application *App) VerifyBackup(recoveryKey string) (security.BackupVerifyResult, error) {
	return application.services.VerifyBackupDialog(application.ctx, recoveryKey)
}

func (application *App) RestoreBackup(recoveryKey string, confirmReplace bool) (security.BackupManifest, error) {
	return application.services.RestoreBackupDialog(application.ctx, recoveryKey, confirmReplace)
}

func (application *App) SaveRecoveryKit(pin string) error {
	return application.services.SaveRecoveryKitDialog(application.ctx, pin)
}

func (application *App) SaveRecoveryKitFromKey(recoveryKey string) error {
	return application.services.SaveRecoveryKitFromKeyDialog(application.ctx, recoveryKey)
}

func (application *App) GetAppVersion() string {
	return application.services.GetAppVersion()
}

func (application *App) GetDateFormatPattern() string {
	return service.DateFormatPattern()
}

func (application *App) CheckForUpdates() (domain.UpdateCheckResult, error) {
	return application.services.CheckForUpdates(application.ctx)
}

func (application *App) InstallUpdate() error {
	kind, err := application.services.InstallUpdate(application.ctx, func(downloaded, total int64) {
		percent := -1.0
		if total > 0 {
			percent = float64(downloaded) / float64(total) * 100
		}
		runtime.EventsEmit(application.ctx, "update:download-progress", map[string]interface{}{
			"downloaded": downloaded,
			"total":      total,
			"percent":    percent,
		})
	})
	if err != nil {
		return err
	}
	// Windows Setup launches its own installer UI; do not relaunch the old binary.
	if kind != update.KindInstaller {
		_ = service.RelaunchCurrentExecutable()
	}
	runtime.Quit(application.ctx)
	return nil
}