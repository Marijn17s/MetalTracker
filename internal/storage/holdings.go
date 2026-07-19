package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"MetalTracker/internal/domain"

	"github.com/google/uuid"
)

func (database *DB) CreateInvestment(request domain.CreateInvestmentRequest, lines []preparedLine) (string, error) {
	purchasedAt, err := time.Parse("2006-01-02", request.PurchasedAt)
	if err != nil {
		return "", fmt.Errorf("invalid purchase date: %w", err)
	}

	tx, err := database.conn.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	investmentID := uuid.NewString()
	now := time.Now().UTC()
	if _, err := tx.Exec(
		`INSERT INTO investments(id, purchased_at, currency, notes, dealer, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		investmentID,
		purchasedAt.Format(time.RFC3339),
		string(request.Currency),
		request.Notes,
		request.Dealer,
		now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}

	for _, line := range lines {
		for index := 0; index < line.Quantity; index++ {
			unitID := uuid.NewString()
			if _, err := tx.Exec(
				`INSERT INTO holding_units(
					id, investment_id, asset_class, metal, form, weight_grams, purity,
					brand, product_name, product_key, currency, purchase_price,
					spot_worth_at_purchase, purchased_at, status, sold_at, sale_price, notes, dealer,
					storage_location, condition, mintage_year, assay_notes, verified_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, ?, ?, ?, ?, ?)`,
				unitID,
				investmentID,
				string(line.AssetClass),
				string(line.Metal),
				string(line.Form),
				line.WeightGrams,
				line.Purity,
				line.Brand,
				line.ProductName,
				line.ProductKey,
				string(request.Currency),
				line.UnitPurchasePrice,
				line.UnitSpotWorth,
				purchasedAt.Format(time.RFC3339),
				string(domain.UnitStatusHeld),
				request.Notes,
				request.Dealer,
				line.StorageLocation,
				line.Condition,
				line.MintageYear,
				"",
				"",
			); err != nil {
				return "", err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return investmentID, nil
}

type preparedLine struct {
	AssetClass        domain.AssetClass
	Metal             domain.MetalSymbol
	Form              domain.Form
	WeightGrams       float64
	Purity            float64
	Brand             string
	ProductName       string
	ProductKey        string
	Quantity          int
	UnitPurchasePrice float64
	UnitSpotWorth     float64
	StorageLocation string
	Condition       string
	MintageYear     int
}

func PrepareLines(inputs []domain.InvestmentLineInput) ([]preparedLine, error) {
	prepared := make([]preparedLine, 0, len(inputs))
	for _, input := range inputs {
		if input.Quantity <= 0 {
			return nil, fmt.Errorf("quantity must be positive")
		}
		if input.TotalPurchasePrice < 0 {
			return nil, fmt.Errorf("purchase price cannot be negative")
		}
		weightGrams, err := domain.ToGrams(input.Weight, input.WeightUnit)
		if err != nil {
			return nil, err
		}
		assetClass := input.AssetClass
		if assetClass == "" {
			assetClass = domain.AssetClassPreciousMetal
		}
		purity := domain.NormalizePurity(input.Purity)
		productKey := domain.ProductKey{
			AssetClass:  assetClass,
			Metal:       input.Metal,
			Form:        input.Form,
			WeightGrams: weightGrams,
			Purity:      purity,
			Brand:       strings.TrimSpace(input.Brand),
			ProductName: strings.TrimSpace(input.ProductName),
		}
		prepared = append(prepared, preparedLine{
			AssetClass:        assetClass,
			Metal:             input.Metal,
			Form:              input.Form,
			WeightGrams:       weightGrams,
			Purity:            purity,
			Brand:             productKey.Brand,
			ProductName:       productKey.ProductName,
			ProductKey:        productKey.String(),
			Quantity:          input.Quantity,
			UnitPurchasePrice: input.TotalPurchasePrice / float64(input.Quantity),
			UnitSpotWorth:     input.TotalSpotWorth / float64(input.Quantity),
			StorageLocation:   strings.TrimSpace(input.StorageLocation),
			Condition:         strings.TrimSpace(input.Condition),
			MintageYear:       input.MintageYear,
		})
	}
	return prepared, nil
}

const unitSelectColumns = `
	id, investment_id, asset_class, metal, form, weight_grams, purity,
	brand, product_name, product_key, currency, purchase_price,
	spot_worth_at_purchase, purchased_at, status, sold_at, sale_price, notes, dealer,
	storage_location, condition, mintage_year, assay_notes, verified_at, deleted_at`

const activeUnitClause = `(deleted_at IS NULL OR deleted_at = '')`

func (database *DB) ListUnits() ([]domain.HoldingUnit, error) {
	rows, err := database.conn.Query(`
		SELECT ` + unitSelectColumns + `
		FROM holding_units
		WHERE ` + activeUnitClause + `
		ORDER BY purchased_at DESC, product_name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUnits(rows)
}

func (database *DB) ListUnitsByProductKey(productKey string) ([]domain.HoldingUnit, error) {
	rows, err := database.conn.Query(`
		SELECT ` + unitSelectColumns + `
		FROM holding_units
		WHERE product_key = ? AND ` + activeUnitClause + `
		ORDER BY purchased_at DESC, id ASC`, productKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUnits(rows)
}

func (database *DB) ListSoldUnits() ([]domain.HoldingUnit, error) {
	rows, err := database.conn.Query(`
		SELECT ` + unitSelectColumns + `
		FROM holding_units
		WHERE status = ? AND ` + activeUnitClause + `
		ORDER BY sold_at DESC, purchased_at DESC, id ASC`,
		string(domain.UnitStatusSold),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUnits(rows)
}

func (database *DB) ListDeletedUnits() ([]domain.HoldingUnit, error) {
	rows, err := database.conn.Query(`
		SELECT ` + unitSelectColumns + `
		FROM holding_units
		WHERE deleted_at IS NOT NULL AND deleted_at != ''
		ORDER BY deleted_at DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUnits(rows)
}

func (database *DB) GetUnit(unitID string) (domain.HoldingUnit, error) {
	row := database.conn.QueryRow(`
		SELECT `+unitSelectColumns+`
		FROM holding_units
		WHERE id = ? AND `+activeUnitClause, unitID)
	return scanUnit(row)
}

func (database *DB) GetUnitIncludingDeleted(unitID string) (domain.HoldingUnit, error) {
	row := database.conn.QueryRow(`
		SELECT `+unitSelectColumns+`
		FROM holding_units WHERE id = ?`, unitID)
	return scanUnit(row)
}

func (database *DB) SellUnit(unitID string, soldAt time.Time, salePrice float64) error {
	result, err := database.conn.Exec(`
		UPDATE holding_units
		SET status = ?, sold_at = ?, sale_price = ?
		WHERE id = ? AND status = ? AND `+activeUnitClause,
		string(domain.UnitStatusSold),
		soldAt.Format(time.RFC3339),
		salePrice,
		unitID,
		string(domain.UnitStatusHeld),
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("unit not found or already sold")
	}
	return nil
}

func (database *DB) UpdateUnit(unit domain.HoldingUnit) error {
	var soldAt any
	var salePrice any
	if unit.Status == domain.UnitStatusSold {
		if unit.SoldAt == "" {
			return fmt.Errorf("sold units require a sale date")
		}
		if unit.SalePrice == nil {
			return fmt.Errorf("sold units require a sale price")
		}
		soldAt = unit.SoldAt
		salePrice = *unit.SalePrice
	}

	result, err := database.conn.Exec(`
		UPDATE holding_units
		SET asset_class = ?, metal = ?, form = ?, weight_grams = ?, purity = ?,
		    brand = ?, product_name = ?, product_key = ?, purchase_price = ?,
		    spot_worth_at_purchase = ?, purchased_at = ?, status = ?,
		    sold_at = ?, sale_price = ?, notes = ?, dealer = ?,
		    storage_location = ?, condition = ?, mintage_year = ?
		WHERE id = ? AND `+activeUnitClause,
		string(unit.AssetClass),
		string(unit.Metal),
		string(unit.Form),
		unit.WeightGrams,
		unit.Purity,
		unit.Brand,
		unit.ProductName,
		unit.ProductKey,
		unit.PurchasePrice,
		unit.SpotWorthAtPurchase,
		unit.PurchasedAt,
		string(unit.Status),
		soldAt,
		salePrice,
		unit.Notes,
		unit.Dealer,
		unit.StorageLocation,
		unit.Condition,
		unit.MintageYear,
		unit.ID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("unit not found")
	}
	return nil
}

func (database *DB) BulkUpdateUnitFields(unitIDs []string, dealer string, storageLocation string, notes string) error {
	if len(unitIDs) == 0 {
		return fmt.Errorf("at least one unit is required")
	}
	tx, err := database.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, unitID := range unitIDs {
		result, execErr := tx.Exec(`
			UPDATE holding_units
			SET dealer = ?, storage_location = ?, notes = ?
			WHERE id = ? AND `+activeUnitClause,
			dealer,
			storageLocation,
			notes,
			unitID,
		)
		if execErr != nil {
			return execErr
		}
		affected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return rowsErr
		}
		if affected == 0 {
			return fmt.Errorf("unit not found: %s", unitID)
		}
	}
	return tx.Commit()
}

func (database *DB) SoftDeleteUnit(unitID string) error {
	result, err := database.conn.Exec(`
		UPDATE holding_units
		SET deleted_at = ?
		WHERE id = ? AND `+activeUnitClause,
		time.Now().UTC().Format(time.RFC3339),
		unitID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("unit not found")
	}
	return nil
}

func (database *DB) RestoreUnit(unitID string) error {
	result, err := database.conn.Exec(`
		UPDATE holding_units
		SET deleted_at = ''
		WHERE id = ? AND deleted_at IS NOT NULL AND deleted_at != ''`,
		unitID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("deleted unit not found")
	}
	return nil
}

func (database *DB) DeleteUnit(unitID string) error {
	unit, err := database.GetUnitIncludingDeleted(unitID)
	if err != nil {
		return err
	}
	result, err := database.conn.Exec(`DELETE FROM holding_units WHERE id = ?`, unitID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("unit not found")
	}
	return database.DeleteInvestmentIfOrphaned(unit.InvestmentID)
}

func (database *DB) DeleteInvestmentIfOrphaned(investmentID string) error {
	if investmentID == "" {
		return nil
	}
	var remaining int
	if err := database.conn.QueryRow(
		`SELECT COUNT(*) FROM holding_units WHERE investment_id = ?`,
		investmentID,
	).Scan(&remaining); err != nil {
		return err
	}
	if remaining > 0 {
		return nil
	}
	_, err := database.conn.Exec(`DELETE FROM investments WHERE id = ?`, investmentID)
	return err
}

func (database *DB) DeleteOrphanInvestments() (int64, error) {
	result, err := database.conn.Exec(`
		DELETE FROM investments
		WHERE id NOT IN (SELECT DISTINCT investment_id FROM holding_units)`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanUnits(rows *sql.Rows) ([]domain.HoldingUnit, error) {
	units := make([]domain.HoldingUnit, 0)
	for rows.Next() {
		unit, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		units = append(units, unit)
	}
	return units, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUnit(row scanner) (domain.HoldingUnit, error) {
	var unit domain.HoldingUnit
	var purchasedAt string
	var soldAt sql.NullString
	var salePrice sql.NullFloat64
	var assetClass, metal, form, currency, status string
	var unusedAssayNotes string
	var unusedVerifiedAt string
	var deletedAt string

	err := row.Scan(
		&unit.ID,
		&unit.InvestmentID,
		&assetClass,
		&metal,
		&form,
		&unit.WeightGrams,
		&unit.Purity,
		&unit.Brand,
		&unit.ProductName,
		&unit.ProductKey,
		&currency,
		&unit.PurchasePrice,
		&unit.SpotWorthAtPurchase,
		&purchasedAt,
		&status,
		&soldAt,
		&salePrice,
		&unit.Notes,
		&unit.Dealer,
		&unit.StorageLocation,
		&unit.Condition,
		&unit.MintageYear,
		&unusedAssayNotes,
		&unusedVerifiedAt,
		&deletedAt,
	)
	if err != nil {
		return unit, err
	}

	unit.AssetClass = domain.AssetClass(assetClass)
	unit.Metal = domain.MetalSymbol(metal)
	unit.Form = domain.Form(form)
	unit.Currency = domain.Currency(currency)
	unit.Status = domain.UnitStatus(status)
	unit.PurchasedAt = purchasedAt
	unit.DeletedAt = deletedAt
	if soldAt.Valid {
		unit.SoldAt = soldAt.String
	}
	if salePrice.Valid {
		value := salePrice.Float64
		unit.SalePrice = &value
	}
	return unit, nil
}
