package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"MetalTracker/internal/domain"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
	path string
}

type migration struct {
	version int
	name    string
	up      func(conn *sql.DB) error
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = conn.Close()
		return nil, err
	}
	database := &DB{conn: conn, path: path}
	if err := database.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return database, nil
}

func OpenPriceCache(path string) (*DB, error) {
	return Open(path)
}

func (database *DB) Conn() *sql.DB {
	return database.conn
}

func (database *DB) Close() error {
	if database == nil || database.conn == nil {
		return nil
	}
	return database.conn.Close()
}

func (database *DB) migrate() error {
	if _, err := database.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	currentVersion, err := database.currentSchemaVersion()
	if err != nil {
		return err
	}

	migrations := allMigrations()
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	for _, item := range migrations {
		if item.version <= currentVersion {
			continue
		}
		if err := item.up(database.conn); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", item.version, item.name, err)
		}
		if _, err := database.conn.Exec(
			`INSERT INTO schema_migrations(version) VALUES (?)`,
			item.version,
		); err != nil {
			return fmt.Errorf("record migration %d: %w", item.version, err)
		}
		currentVersion = item.version
	}
	return nil
}

func (database *DB) currentSchemaVersion() (int, error) {
	var version sql.NullInt64
	err := database.conn.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version)
	if err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func allMigrations() []migration {
	return []migration{
		{version: 1, name: "initial_schema", up: migrateV1},
		{version: 2, name: "settings_defaults_v2", up: migrateV2},
		{version: 3, name: "inventory_enrichment", up: migrateV3},
		{version: 4, name: "soft_delete_units", up: migrateV4},
	}
}

func migrateV1(conn *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS investments (
			id TEXT PRIMARY KEY,
			purchased_at TEXT NOT NULL,
			currency TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			dealer TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS holding_units (
			id TEXT PRIMARY KEY,
			investment_id TEXT NOT NULL,
			asset_class TEXT NOT NULL,
			metal TEXT NOT NULL,
			form TEXT NOT NULL,
			weight_grams REAL NOT NULL,
			purity REAL NOT NULL,
			brand TEXT NOT NULL DEFAULT '',
			product_name TEXT NOT NULL DEFAULT '',
			product_key TEXT NOT NULL,
			currency TEXT NOT NULL,
			purchase_price REAL NOT NULL,
			spot_worth_at_purchase REAL NOT NULL,
			purchased_at TEXT NOT NULL,
			status TEXT NOT NULL,
			sold_at TEXT,
			sale_price REAL,
			notes TEXT NOT NULL DEFAULT '',
			dealer TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (investment_id) REFERENCES investments(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_holding_units_product_key ON holding_units(product_key)`,
		`CREATE INDEX IF NOT EXISTS idx_holding_units_status ON holding_units(status)`,
		`CREATE INDEX IF NOT EXISTS idx_holding_units_metal ON holding_units(metal)`,
		`CREATE TABLE IF NOT EXISTS price_quotes (
			quote_date TEXT NOT NULL,
			base_currency TEXT NOT NULL,
			symbol TEXT NOT NULL,
			unit TEXT NOT NULL,
			price_per_unit REAL NOT NULL,
			fetched_at TEXT NOT NULL,
			PRIMARY KEY (quote_date, base_currency, symbol, unit)
		)`,
	}

	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			return err
		}
	}

	defaults := map[string]string{
		"display_currency":   "EUR",
		"price_source":       string(domain.PriceSourceMiddleman),
		"metalprice_api_key": "",
		"middleman_base_url": domain.DefaultMiddlemanBaseURL,
		"auto_lock_minutes":  "15",
		"spot_price_unit":    "troy_oz",
		"ui_theme":           "dark",
	}
	for key, value := range defaults {
		if _, err := conn.Exec(
			`INSERT OR IGNORE INTO settings(key, value) VALUES (?, ?)`,
			key, value,
		); err != nil {
			return err
		}
	}
	return nil
}

func migrateV2(conn *sql.DB) error {
	// Reserved for historical installs that already recorded version 2.
	// Newer settings keys should use later migration versions.
	return nil
}

func migrateV3(conn *sql.DB) error {
	statements := []string{
		`ALTER TABLE holding_units ADD COLUMN storage_location TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE holding_units ADD COLUMN condition TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE holding_units ADD COLUMN mintage_year INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE holding_units ADD COLUMN assay_notes TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE holding_units ADD COLUMN verified_at TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS attachments (
			id TEXT PRIMARY KEY,
			owner_type TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			filename TEXT NOT NULL,
			content_type TEXT NOT NULL,
			relative_path TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_owner ON attachments(owner_type, owner_id)`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func migrateV4(conn *sql.DB) error {
	statements := []string{
		`ALTER TABLE holding_units ADD COLUMN deleted_at TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_holding_units_deleted_at ON holding_units(deleted_at)`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}
