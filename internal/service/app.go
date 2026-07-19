package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/domain"
	"MetalTracker/internal/price"
	"MetalTracker/internal/security"
	"MetalTracker/internal/storage"
	"MetalTracker/internal/update"
)

type AppServices struct {
	mu            sync.Mutex
	dataDir       string
	vault         *security.Vault
	vaultDB       *storage.DB
	priceDB       *storage.DB
	metalClient   *price.MetalpriceAPIClient
	middleman     *price.MiddlemanProvider
	cachedPrices  *price.CachedProvider
	autoLockStop  chan struct{}
	pendingUpdate *update.Pending
}

func NewAppServices() (*AppServices, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(configDir, "MetalTracker")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}

	vault, err := security.NewVault(dataDir)
	if err != nil {
		return nil, err
	}

	priceDBPath := filepath.Join(dataDir, "prices.db")
	priceDB, err := storage.OpenPriceCache(priceDBPath)
	if err != nil {
		return nil, err
	}

	metalClient := price.NewMetalpriceAPIClient("")
	middleman := price.NewMiddlemanProvider("")
	cached := price.NewCachedProvider(metalClient, priceDB)

	services := &AppServices{
		dataDir:      dataDir,
		vault:        vault,
		priceDB:      priceDB,
		metalClient:  metalClient,
		middleman:    middleman,
		cachedPrices: cached,
	}
	return services, nil
}

func (services *AppServices) Close() {
	services.mu.Lock()
	defer services.mu.Unlock()
	services.stopAutoLockLocked()
	if services.vaultDB != nil {
		_ = services.vault.Persist()
		_ = services.vaultDB.Close()
		services.vaultDB = nil
	}
	_ = services.vault.Lock()
	if services.priceDB != nil {
		_ = services.priceDB.Close()
	}
}

func (services *AppServices) VaultExists() bool {
	return services.vault.Exists()
}

func (services *AppServices) IsUnlocked() bool {
	return services.vault.IsUnlocked()
}

func (services *AppServices) SetupVault(pin string) (*security.SetupResult, error) {
	services.mu.Lock()
	defer services.mu.Unlock()

	result, err := services.vault.Setup(pin)
	if err != nil {
		return nil, mapError(err)
	}
	if err := services.vault.EnsurePlainDBForSetup(); err != nil {
		return nil, mapError(err)
	}
	if err := services.openVaultDBLocked(); err != nil {
		return nil, mapError(err)
	}
	// Apply seeded defaults (e.g. Middleman URL/source) before the first price fetch.
	if err := services.applySettingsToPriceLayerLocked(); err != nil {
		return nil, mapError(err)
	}
	if err := services.vault.Persist(); err != nil {
		return nil, mapError(err)
	}
	services.startAutoLockLocked()
	return result, nil
}

func (services *AppServices) Unlock(pin string) error {
	services.mu.Lock()
	defer services.mu.Unlock()

	if err := services.vault.Unlock(pin); err != nil {
		return mapError(err)
	}
	if err := services.openVaultDBLocked(); err != nil {
		_ = services.vault.Lock()
		return mapError(err)
	}
	if err := services.applySettingsToPriceLayerLocked(); err != nil {
		return err
	}
	_, _ = services.vaultDB.DeleteOrphanInvestments()
	services.startAutoLockLocked()
	return nil
}

func (services *AppServices) Lock() error {
	services.mu.Lock()
	defer services.mu.Unlock()
	services.stopAutoLockLocked()
	if services.vaultDB != nil {
		_ = services.vault.Persist()
		_ = services.vaultDB.Close()
		services.vaultDB = nil
	}
	return services.vault.Lock()
}

func (services *AppServices) RecoverWithKey(recoveryKey string, newPIN string) error {
	services.mu.Lock()
	defer services.mu.Unlock()

	if err := services.vault.RecoverWithKey(recoveryKey, newPIN); err != nil {
		return mapError(err)
	}
	if err := services.openVaultDBLocked(); err != nil {
		_ = services.vault.Lock()
		return mapError(err)
	}
	if err := services.applySettingsToPriceLayerLocked(); err != nil {
		return mapError(err)
	}
	_, _ = services.vaultDB.DeleteOrphanInvestments()
	if err := services.vault.Persist(); err != nil {
		return mapError(err)
	}
	services.startAutoLockLocked()
	return nil
}

func (services *AppServices) ChangePIN(currentPIN string, newPIN string) error {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	return mapError(services.vault.ChangePIN(currentPIN, newPIN))
}

func (services *AppServices) TouchActivity() {
	services.vault.TouchActivity()
}

func (services *AppServices) openVaultDBLocked() error {
	if services.vaultDB != nil {
		_ = services.vaultDB.Close()
		services.vaultDB = nil
	}
	database, err := storage.Open(services.vault.PlainDBPath())
	if err != nil {
		return err
	}
	services.vaultDB = database
	return nil
}

func (services *AppServices) requireUnlockedLocked() error {
	if !services.vault.IsUnlocked() || services.vaultDB == nil {
		return security.ErrVaultLocked
	}
	services.vault.TouchActivity()
	return nil
}

func (services *AppServices) applySettingsToPriceLayerLocked() error {
	if services.vaultDB == nil {
		return security.ErrVaultLocked
	}
	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		return err
	}
	services.metalClient.SetAPIKey(settings.MetalpriceAPIKey)
	services.middleman.SetBaseURL(settings.MiddlemanBaseURL)
	services.vault.SetAutoLockAfter(time.Duration(settings.AutoLockMinutes) * time.Minute)

	switch settings.PriceSource {
	case domain.PriceSourceMiddleman:
		services.cachedPrices.SetInner(services.middleman)
	default:
		services.cachedPrices.SetInner(services.metalClient)
	}
	return nil
}

func (services *AppServices) priceProvider() price.Provider {
	return services.cachedPrices
}

func (services *AppServices) startAutoLockLocked() {
	services.stopAutoLockLocked()
	services.autoLockStop = make(chan struct{})
	stop := services.autoLockStop
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if services.vault.CheckAutoLock() {
					services.mu.Lock()
					if services.vaultDB != nil {
						_ = services.vaultDB.Close()
						services.vaultDB = nil
					}
					services.mu.Unlock()
					return
				}
			}
		}
	}()
}

func (services *AppServices) stopAutoLockLocked() {
	if services.autoLockStop != nil {
		close(services.autoLockStop)
		services.autoLockStop = nil
	}
}

func (services *AppServices) withDB(action func(database *storage.DB) error) error {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return err
	}
	if err := action(services.vaultDB); err != nil {
		return err
	}
	return services.vault.Persist()
}

func (services *AppServices) GetSettings() (domain.AppSettings, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.AppSettings{}, mapError(err)
	}
	return services.vaultDB.GetSettings()
}

func (services *AppServices) UpdateSettings(settings domain.AppSettings) error {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	if settings.DisplayCurrency != domain.CurrencyEUR &&
		settings.DisplayCurrency != domain.CurrencyUSD &&
		settings.DisplayCurrency != domain.CurrencyCHF {
		return apperr.Validation("unsupported display currency")
	}
	if settings.SpotPriceUnit != domain.SpotPriceUnitGram &&
		settings.SpotPriceUnit != domain.SpotPriceUnitTroyOz &&
		settings.SpotPriceUnit != domain.SpotPriceUnitKilogram &&
		settings.SpotPriceUnit != "" {
		return apperr.Validation("unsupported spot price unit")
	}
	if settings.SpotPriceUnit == "" {
		settings.SpotPriceUnit = domain.SpotPriceUnitTroyOz
	}
	if settings.AutoLockMinutes <= 0 {
		settings.AutoLockMinutes = 15
	}
	if err := services.vaultDB.UpdateSettings(settings); err != nil {
		return mapError(err)
	}
	if err := services.applySettingsToPriceLayerLocked(); err != nil {
		return mapError(err)
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) GetLatestPrices(ctx context.Context) (domain.SpotQuote, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.SpotQuote{}, mapError(err)
	}
	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		return domain.SpotQuote{}, mapError(err)
	}
	quote, err := services.priceProvider().Latest(
		ctx,
		string(settings.DisplayCurrency),
		price.ValuationSymbols(settings.DisplayCurrency),
	)
	if err != nil {
		return domain.SpotQuote{}, mapError(err)
	}
	return price.ToSpotQuote(quote), nil
}
