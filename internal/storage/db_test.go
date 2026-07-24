package storage

import (
	"path/filepath"
	"testing"
	"time"

	"MetalTracker/internal/domain"
)

func TestMigrationsAndOrphanCleanup(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(filepath.Join(dir, "vault.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	version, err := database.currentSchemaVersion()
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if version != 5 {
		t.Fatalf("expected schema version 5, got %d", version)
	}

	// Re-open should be idempotent.
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = Open(filepath.Join(dir, "vault.db"))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer database.Close()

	prepared, err := PrepareLines([]domain.InvestmentLineInput{{
		Metal:              domain.MetalGold,
		Form:               domain.FormBar,
		Weight:             1,
		WeightUnit:         domain.WeightUnitTroyOz,
		Purity:             999.9,
		Quantity:           1,
		TotalPurchasePrice: 2000,
		TotalSpotWorth:     1900,
		SpotWorthProvided:  true,
	}})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	investmentID, err := database.CreateInvestment(domain.CreateInvestmentRequest{
		PurchasedAt: time.Now().UTC().Format("2006-01-02"),
		Currency:    domain.CurrencyEUR,
	}, prepared)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	units, err := database.ListUnits()
	if err != nil || len(units) != 1 {
		t.Fatalf("units: %v len=%d", err, len(units))
	}

	if err := database.DeleteUnit(units[0].ID); err != nil {
		t.Fatalf("delete unit: %v", err)
	}

	var investmentCount int
	if err := database.conn.QueryRow(`SELECT COUNT(*) FROM investments WHERE id = ?`, investmentID).Scan(&investmentCount); err != nil {
		t.Fatal(err)
	}
	if investmentCount != 0 {
		t.Fatal("expected orphan investment cleanup")
	}

	// Explicit orphan cleanup path.
	if _, err := database.conn.Exec(
		`INSERT INTO investments(id, purchased_at, currency, notes, dealer, created_at)
		 VALUES ('orphan', ?, 'EUR', '', '', ?)`,
		time.Now().UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		t.Fatal(err)
	}
	deleted, err := database.DeleteOrphanInvestments()
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 orphan deleted, got %d", deleted)
	}
}
