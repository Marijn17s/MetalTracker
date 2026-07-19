package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const metals = "XAU,XAG,XPT,XPD"

// Price is one metal quote snapshot with multi-currency kg prices.
type Price struct {
	ID       int64
	DateTime time.Time
	Metal    string
	EUR      float64
	USD      float64
	CHF      float64
	GBP      float64
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (store *Store) Close() error {
	return store.db.Close()
}

func (store *Store) migrate() error {
	_, err := store.db.Exec(`
		CREATE TABLE IF NOT EXISTS price (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			datetime TEXT NOT NULL,
			metal TEXT NOT NULL,
			eur REAL NOT NULL,
			usd REAL NOT NULL,
			chf REAL NOT NULL,
			gbp REAL NOT NULL,
			UNIQUE(metal, datetime)
		);
		CREATE INDEX IF NOT EXISTS idx_price_metal_datetime ON price(metal, datetime DESC);
	`)
	return err
}

// InsertHourly appends a live hourly snapshot. Ignores duplicate (metal, datetime).
func (store *Store) InsertHourly(row Price) error {
	_, err := store.db.Exec(
		`INSERT OR IGNORE INTO price (datetime, metal, eur, usd, chf, gbp) VALUES (?, ?, ?, ?, ?, ?)`,
		row.DateTime.UTC().Format(time.RFC3339),
		strings.ToUpper(row.Metal),
		row.EUR, row.USD, row.CHF, row.GBP,
	)
	return err
}

// InsertHistoricalIfMissing stores one daily row for a past calendar day (noon UTC).
func (store *Store) InsertHistoricalIfMissing(row Price) error {
	day := row.DateTime.UTC().Format("2006-01-02")
	var count int
	err := store.db.QueryRow(
		`SELECT COUNT(1) FROM price WHERE metal = ? AND date(datetime) = date(?)`,
		strings.ToUpper(row.Metal), day,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	stamp := time.Date(row.DateTime.UTC().Year(), row.DateTime.UTC().Month(), row.DateTime.UTC().Day(), 12, 0, 0, 0, time.UTC)
	_, err = store.db.Exec(
		`INSERT OR IGNORE INTO price (datetime, metal, eur, usd, chf, gbp) VALUES (?, ?, ?, ?, ?, ?)`,
		stamp.Format(time.RFC3339),
		strings.ToUpper(row.Metal),
		row.EUR, row.USD, row.CHF, row.GBP,
	)
	return err
}

// Latest returns the newest row for each requested metal.
func (store *Store) Latest(metals []string) (map[string]Price, error) {
	result := make(map[string]Price, len(metals))
	for _, metal := range metals {
		row, ok, err := store.latestMetal(metal)
		if err != nil {
			return nil, err
		}
		if ok {
			result[metal] = row
		}
	}
	return result, nil
}

func (store *Store) latestMetal(metal string) (Price, bool, error) {
	row := store.db.QueryRow(
		`SELECT id, datetime, metal, eur, usd, chf, gbp FROM price WHERE metal = ? ORDER BY datetime DESC LIMIT 1`,
		strings.ToUpper(metal),
	)
	return scanPrice(row)
}

// LatestOnDay returns the last snapshot on the given calendar day (UTC) for each metal.
func (store *Store) LatestOnDay(day time.Time, metals []string) (map[string]Price, error) {
	dayKey := day.UTC().Format("2006-01-02")
	result := make(map[string]Price, len(metals))
	for _, metal := range metals {
		row := store.db.QueryRow(
			`SELECT id, datetime, metal, eur, usd, chf, gbp FROM price
			 WHERE metal = ? AND date(datetime) = date(?)
			 ORDER BY datetime DESC LIMIT 1`,
			strings.ToUpper(metal), dayKey,
		)
		priceRow, ok, err := scanPrice(row)
		if err != nil {
			return nil, err
		}
		if ok {
			result[metal] = priceRow
		}
	}
	return result, nil
}

// HasDay reports whether any row exists for metal on that calendar day.
func (store *Store) HasDay(metal string, day time.Time) (bool, error) {
	var count int
	err := store.db.QueryRow(
		`SELECT COUNT(1) FROM price WHERE metal = ? AND date(datetime) = date(?)`,
		strings.ToUpper(metal), day.UTC().Format("2006-01-02"),
	).Scan(&count)
	return count > 0, err
}

// DaysInRange returns, for each calendar day in [from, to], the last row that day per metal.
func (store *Store) DaysInRange(from, to time.Time, metalsList []string) (map[string]map[string]Price, error) {
	// dateKey -> metal -> price
	result := make(map[string]map[string]Price)
	fromKey := from.UTC().Format("2006-01-02")
	toKey := to.UTC().Format("2006-01-02")
	placeholders := make([]string, 0, len(metalsList))
	args := make([]any, 0, len(metalsList)+2)
	for _, metal := range metalsList {
		placeholders = append(placeholders, "?")
		args = append(args, strings.ToUpper(metal))
	}
	args = append(args, fromKey, toKey)

	query := fmt.Sprintf(`
		SELECT id, datetime, metal, eur, usd, chf, gbp FROM price
		WHERE metal IN (%s) AND date(datetime) BETWEEN date(?) AND date(?)
		ORDER BY datetime ASC
	`, strings.Join(placeholders, ","))

	rows, err := store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var priceRow Price
		var stamp string
		if err := rows.Scan(&priceRow.ID, &stamp, &priceRow.Metal, &priceRow.EUR, &priceRow.USD, &priceRow.CHF, &priceRow.GBP); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339, stamp)
		if err != nil {
			parsed, _ = time.Parse("2006-01-02 15:04:05", stamp)
		}
		priceRow.DateTime = parsed.UTC()
		dayKey := priceRow.DateTime.Format("2006-01-02")
		if result[dayKey] == nil {
			result[dayKey] = make(map[string]Price)
		}
		// Ascending scan: last write wins per metal/day.
		result[dayKey][priceRow.Metal] = priceRow
	}
	return result, rows.Err()
}

func scanPrice(row *sql.Row) (Price, bool, error) {
	var priceRow Price
	var stamp string
	err := row.Scan(&priceRow.ID, &stamp, &priceRow.Metal, &priceRow.EUR, &priceRow.USD, &priceRow.CHF, &priceRow.GBP)
	if err == sql.ErrNoRows {
		return Price{}, false, nil
	}
	if err != nil {
		return Price{}, false, err
	}
	parsed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		parsed, err = time.Parse("2006-01-02 15:04:05", stamp)
		if err != nil {
			return Price{}, false, err
		}
	}
	priceRow.DateTime = parsed.UTC()
	return priceRow, true, nil
}

// DefaultMetals returns the metals the poller tracks.
func DefaultMetals() []string {
	return strings.Split(metals, ",")
}
