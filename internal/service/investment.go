package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/domain"
	"MetalTracker/internal/price"
	"MetalTracker/internal/storage"
)

func (services *AppServices) CreateInvestment(ctx context.Context, request domain.CreateInvestmentRequest) (string, error) {
	if len(request.Lines) == 0 {
		return "", apperr.Validation("at least one investment line is required")
	}
	if request.Currency == "" {
		request.Currency = domain.CurrencyEUR
	}

	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return "", mapError(err)
	}

	lines := make([]domain.InvestmentLineInput, len(request.Lines))
	copy(lines, request.Lines)

	purchasedAt, err := time.Parse("2006-01-02", request.PurchasedAt)
	if err != nil {
		return "", apperr.Validation("invalid purchase date")
	}

	for index := range lines {
		line := &lines[index]
		if line.Metal != domain.MetalGold && line.Metal != domain.MetalSilver {
			return "", apperr.Validation("unsupported metal: " + string(line.Metal))
		}
		if line.Form == "" {
			line.Form = domain.FormBar
		}
		if line.Purity <= 0 {
			line.Purity = 999.9
		}
		if !line.SpotWorthProvided || line.TotalSpotWorth <= 0 {
			quote, quoteErr := services.priceProvider().Historical(
				ctx,
				purchasedAt,
				string(request.Currency),
				[]string{string(line.Metal)},
			)
			if quoteErr != nil {
				quote, quoteErr = services.priceProvider().Latest(
					ctx,
					string(request.Currency),
					[]string{string(line.Metal)},
				)
			}
			if quoteErr != nil {
				return "", mapError(quoteErr)
			}
			spotPerKg, ok := price.MetalSpotPerKg(quote, line.Metal)
			if !ok {
				return "", apperr.PriceUnavailable("missing spot price for " + string(line.Metal))
			}
			weightGrams, weightErr := domain.ToGrams(line.Weight, line.WeightUnit)
			if weightErr != nil {
				return "", apperr.Validation(weightErr.Error())
			}
			unitWorth := price.SpotWorthForUnit(spotPerKg, weightGrams, domain.NormalizePurity(line.Purity))
			line.TotalSpotWorth = unitWorth * float64(line.Quantity)
			line.SpotWorthProvided = true
		}
	}

	prepared, err := storage.PrepareLines(lines)
	if err != nil {
		return "", apperr.Validation(err.Error())
	}

	investmentID, err := services.vaultDB.CreateInvestment(request, prepared)
	if err != nil {
		return "", mapError(err)
	}
	if err := services.vault.Persist(); err != nil {
		return "", mapError(err)
	}
	return investmentID, nil
}

func (services *AppServices) ListGroupedHoldings(ctx context.Context, filter domain.HoldingsFilter) ([]domain.GroupedHolding, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return nil, mapError(err)
	}

	units, err := services.vaultDB.ListUnits()
	if err != nil {
		return nil, mapError(err)
	}

	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	groups := map[string]*domain.GroupedHolding{}
	orderedProductKeys := make([]string, 0)

	for _, unit := range units {
		if unit.Status != domain.UnitStatusHeld {
			continue
		}
		if !domain.UnitMatchesHoldingsFilter(unit, filter) {
			continue
		}

		valued := valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency)

		group, exists := groups[unit.ProductKey]
		if !exists {
			group = &domain.GroupedHolding{
				ProductKey:      unit.ProductKey,
				AssetClass:      unit.AssetClass,
				Metal:           unit.Metal,
				Form:            unit.Form,
				WeightGrams:     unit.WeightGrams,
				Purity:          unit.Purity,
				Brand:           unit.Brand,
				ProductName:     unit.ProductName,
				Currency:        unit.Currency,
				DisplayCurrency: valuation.settings.DisplayCurrency,
			}
			groups[unit.ProductKey] = group
			orderedProductKeys = append(orderedProductKeys, unit.ProductKey)
		}

		group.TotalCount++
		group.HeldCount++
		group.TotalWeightGrams += unit.WeightGrams
		group.TotalFineWeightGrams += valued.FineWeightGrams
		group.TotalPurchasePrice += valued.PurchasePrice
		group.TotalCurrentWorth += valued.CurrentSpotWorth
		group.TotalProfit += valued.TotalProfit
		group.TotalUnrealizedProfit += valued.TotalProfit
		group.HeldPurchasePrice += valued.PurchasePrice
		group.HeldFineWeightGrams += valued.FineWeightGrams
		if valued.ValuationApproximate {
			group.ValuationApproximate = true
		}
	}

	result := make([]domain.GroupedHolding, 0, len(orderedProductKeys))
	for _, productKey := range orderedProductKeys {
		group := groups[productKey]
		group.TotalProfitPct = domain.Percentage(group.TotalProfit, group.TotalPurchasePrice)
		group.AvgCostPerKgFine = domain.AvgCostPerKgFine(group.TotalPurchasePrice, group.TotalFineWeightGrams)
		group.BreakEvenSpotPerKg = domain.BreakEvenSpotPerKg(group.HeldPurchasePrice, group.HeldFineWeightGrams)
		result = append(result, *group)
	}
	return result, nil
}

func (services *AppServices) GetHoldingsFilterOptions() (domain.HoldingsFilterOptions, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.HoldingsFilterOptions{}, mapError(err)
	}

	units, err := services.vaultDB.ListUnits()
	if err != nil {
		return domain.HoldingsFilterOptions{}, mapError(err)
	}

	brandSet := map[string]struct{}{}
	locationSet := map[string]struct{}{}
	for _, unit := range units {
		brand := strings.TrimSpace(unit.Brand)
		if brand == "" {
			brand = "Unknown"
		}
		brandSet[brand] = struct{}{}
		location := strings.TrimSpace(unit.StorageLocation)
		if location == "" {
			location = "Unset"
		}
		locationSet[location] = struct{}{}
	}

	options := domain.HoldingsFilterOptions{
		Brands:    make([]string, 0, len(brandSet)),
		Locations: make([]string, 0, len(locationSet)),
	}
	for brand := range brandSet {
		options.Brands = append(options.Brands, brand)
	}
	for location := range locationSet {
		options.Locations = append(options.Locations, location)
	}
	sort.Strings(options.Brands)
	sort.Strings(options.Locations)
	return options, nil
}

func (services *AppServices) ListUnitsInGroup(ctx context.Context, productKey string) ([]domain.UnitValuation, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return nil, mapError(err)
	}

	units, err := services.vaultDB.ListUnitsByProductKey(productKey)
	if err != nil {
		return nil, mapError(err)
	}

	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	valuations := make([]domain.UnitValuation, 0, len(units))
	for _, unit := range units {
		if unit.Status != domain.UnitStatusHeld {
			continue
		}
		valuations = append(valuations, valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency))
	}
	return valuations, nil
}

func (services *AppServices) GetUnit(ctx context.Context, unitID string) (domain.UnitValuation, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.UnitValuation{}, mapError(err)
	}

	unit, err := services.vaultDB.GetUnit(unitID)
	if err != nil {
		return domain.UnitValuation{}, apperr.NotFound("unit not found")
	}

	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return domain.UnitValuation{}, mapError(err)
	}

	return valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency), nil
}

func (services *AppServices) SellUnit(request domain.SellUnitRequest) error {
	if request.SalePrice < 0 {
		return apperr.Validation("sale price cannot be negative")
	}
	soldAt, err := time.Parse("2006-01-02", request.SoldAt)
	if err != nil {
		return apperr.Validation("invalid sale date")
	}

	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	if err := services.vaultDB.SellUnit(request.UnitID, soldAt, request.SalePrice); err != nil {
		return apperr.NotFound(err.Error())
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) SellUnits(request domain.SellUnitsRequest) error {
	if len(request.Units) == 0 {
		return apperr.Validation("at least one unit is required")
	}
	for _, item := range request.Units {
		if item.UnitID == "" {
			return apperr.Validation("unit id is required")
		}
		if item.SalePrice < 0 {
			return apperr.Validation("sale price cannot be negative")
		}
		if _, err := time.Parse("2006-01-02", item.SoldAt); err != nil {
			return apperr.Validation("invalid sale date")
		}
	}

	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	for _, item := range request.Units {
		soldAt, _ := time.Parse("2006-01-02", item.SoldAt)
		if err := services.vaultDB.SellUnit(item.UnitID, soldAt, item.SalePrice); err != nil {
			return apperr.NotFound(err.Error())
		}
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) BulkUpdateHoldingUnits(request domain.BulkUpdateHoldingUnitsRequest) error {
	if len(request.UnitIDs) == 0 {
		return apperr.Validation("at least one unit is required")
	}
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	if err := services.vaultDB.BulkUpdateUnitFields(
		request.UnitIDs,
		strings.TrimSpace(request.Dealer),
		strings.TrimSpace(request.StorageLocation),
		request.Notes,
	); err != nil {
		return apperr.NotFound(err.Error())
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) UpdateHoldingUnit(request domain.UpdateHoldingUnitRequest) error {
	if request.UnitID == "" {
		return apperr.Validation("unit id is required")
	}
	if request.Metal != domain.MetalGold && request.Metal != domain.MetalSilver {
		return apperr.Validation("unsupported metal: " + string(request.Metal))
	}
	if request.PurchasePrice < 0 || request.SpotWorthAtPurchase < 0 {
		return apperr.Validation("prices cannot be negative")
	}
	purchasedAt, err := time.Parse("2006-01-02", request.PurchasedAt)
	if err != nil {
		return apperr.Validation("invalid purchase date")
	}
	weightGrams, err := domain.ToGrams(request.Weight, request.WeightUnit)
	if err != nil {
		return apperr.Validation(err.Error())
	}
	if request.Form == "" {
		request.Form = domain.FormBar
	}
	purity := domain.NormalizePurity(request.Purity)
	if request.Purity <= 0 {
		purity = domain.NormalizePurity(999.9)
	}
	assetClass := domain.AssetClassPreciousMetal
	brand := strings.TrimSpace(request.Brand)
	productName := strings.TrimSpace(request.ProductName)
	productKey := domain.ProductKey{
		AssetClass:  assetClass,
		Metal:       request.Metal,
		Form:        request.Form,
		WeightGrams: weightGrams,
		Purity:      purity,
		Brand:       brand,
		ProductName: productName,
	}.String()

	status := request.Status
	if status == "" {
		status = domain.UnitStatusHeld
	}

	unit := domain.HoldingUnit{
		ID:                  request.UnitID,
		AssetClass:          assetClass,
		Metal:               request.Metal,
		Form:                request.Form,
		WeightGrams:         weightGrams,
		Purity:              purity,
		Brand:               brand,
		ProductName:         productName,
		ProductKey:          productKey,
		PurchasePrice:       request.PurchasePrice,
		SpotWorthAtPurchase: request.SpotWorthAtPurchase,
		PurchasedAt:         purchasedAt.Format(time.RFC3339),
		Status:              status,
		Notes:               request.Notes,
		Dealer:              request.Dealer,
		StorageLocation:     strings.TrimSpace(request.StorageLocation),
		Condition:           strings.TrimSpace(request.Condition),
		MintageYear:         request.MintageYear,
	}

	if status == domain.UnitStatusSold {
		if request.SoldAt == "" {
			return apperr.Validation("sale date is required for sold units")
		}
		soldAt, parseErr := time.Parse("2006-01-02", request.SoldAt)
		if parseErr != nil {
			return apperr.Validation("invalid sale date")
		}
		if request.SalePrice == nil {
			return apperr.Validation("sale price is required for sold units")
		}
		if *request.SalePrice < 0 {
			return apperr.Validation("sale price cannot be negative")
		}
		unit.SoldAt = soldAt.Format(time.RFC3339)
		unit.SalePrice = request.SalePrice
	}

	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	existing, err := services.vaultDB.GetUnit(request.UnitID)
	if err != nil {
		return apperr.NotFound("unit not found")
	}
	unit.InvestmentID = existing.InvestmentID
	unit.Currency = existing.Currency
	if err := services.vaultDB.UpdateUnit(unit); err != nil {
		return mapError(err)
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) DeleteHoldingUnit(unitID string) error {
	if unitID == "" {
		return apperr.Validation("unit id is required")
	}
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	if err := services.vaultDB.SoftDeleteUnit(unitID); err != nil {
		return apperr.NotFound(err.Error())
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) RestoreHoldingUnit(unitID string) error {
	if unitID == "" {
		return apperr.Validation("unit id is required")
	}
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	if err := services.vaultDB.RestoreUnit(unitID); err != nil {
		return apperr.NotFound(err.Error())
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) PurgeHoldingUnit(unitID string) error {
	if unitID == "" {
		return apperr.Validation("unit id is required")
	}
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return mapError(err)
	}
	existing, err := services.vaultDB.GetUnitIncludingDeleted(unitID)
	if err != nil {
		return apperr.NotFound(err.Error())
	}
	if existing.DeletedAt == "" {
		return apperr.Validation("only soft-deleted units can be purged")
	}
	if err := services.removeAttachmentsForOwnerLocked(domain.AttachmentOwnerUnit, unitID); err != nil {
		return mapError(err)
	}
	if err := services.vaultDB.DeleteUnit(unitID); err != nil {
		return apperr.NotFound(err.Error())
	}
	remaining, listErr := services.vaultDB.ListUnits()
	if listErr == nil {
		investmentStillHasUnits := false
		for _, unit := range remaining {
			if unit.InvestmentID == existing.InvestmentID {
				investmentStillHasUnits = true
				break
			}
		}
		if !investmentStillHasUnits {
			deletedRemaining, deletedErr := services.vaultDB.ListDeletedUnits()
			if deletedErr == nil {
				for _, unit := range deletedRemaining {
					if unit.InvestmentID == existing.InvestmentID {
						investmentStillHasUnits = true
						break
					}
				}
			}
		}
		if !investmentStillHasUnits {
			_ = services.removeAttachmentsForOwnerLocked(domain.AttachmentOwnerInvestment, existing.InvestmentID)
		}
	}
	return mapError(services.vault.Persist())
}

func (services *AppServices) ListRecentlyDeleted() ([]domain.UnitValuation, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return nil, mapError(err)
	}
	units, err := services.vaultDB.ListDeletedUnits()
	if err != nil {
		return nil, mapError(err)
	}
	valuation, err := services.loadValuationContext(context.Background())
	if err != nil {
		return nil, mapError(err)
	}
	result := make([]domain.UnitValuation, 0, len(units))
	for _, unit := range units {
		valued := valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency)
		result = append(result, valued)
	}
	return result, nil
}

func (services *AppServices) ListSoldArchive(ctx context.Context, filter domain.HoldingsFilter) ([]domain.UnitValuation, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return nil, mapError(err)
	}
	units, err := services.vaultDB.ListSoldUnits()
	if err != nil {
		return nil, mapError(err)
	}
	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	result := make([]domain.UnitValuation, 0, len(units))
	for _, unit := range units {
		if !domain.UnitMatchesHoldingsFilter(unit, filter) {
			continue
		}
		valued := valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency)
		result = append(result, valued)
	}
	return result, nil
}

func (services *AppServices) ListDealerSummaries(ctx context.Context) ([]domain.DealerSummary, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return nil, mapError(err)
	}

	units, err := services.vaultDB.ListUnits()
	if err != nil {
		return nil, mapError(err)
	}
	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	type accumulator struct {
		purchaseIDs map[string]struct{}
		unitCount   int
		totalSpent  float64
	}
	byDealer := map[string]*accumulator{}
	for _, unit := range units {
		name := strings.TrimSpace(unit.Dealer)
		if name == "" {
			continue
		}
		entry, exists := byDealer[name]
		if !exists {
			entry = &accumulator{purchaseIDs: map[string]struct{}{}}
			byDealer[name] = entry
		}
		entry.unitCount++
		entry.purchaseIDs[unit.InvestmentID] = struct{}{}
		valued := valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency)
		entry.totalSpent += valued.PurchasePrice
	}

	result := make([]domain.DealerSummary, 0, len(byDealer))
	for name, entry := range byDealer {
		result = append(result, domain.DealerSummary{
			Name:          name,
			PurchaseCount: len(entry.purchaseIDs),
			UnitCount:     entry.unitCount,
			TotalSpent:    entry.totalSpent,
			Currency:      valuation.settings.DisplayCurrency,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}
