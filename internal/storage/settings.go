package storage

import (
	"database/sql"
	"strconv"

	"MetalTracker/internal/domain"
)

func (database *DB) GetSettings() (domain.AppSettings, error) {
	settings := domain.AppSettings{
		DisplayCurrency:  domain.CurrencyEUR,
		PriceSource:      domain.PriceSourceMiddleman,
		MiddlemanBaseURL: domain.DefaultMiddlemanBaseURL,
		AutoLockMinutes:  15,
		SpotPriceUnit:    domain.SpotPriceUnitTroyOz,
		UITheme:          domain.UIThemeDark,
	}
	rows, err := database.conn.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return settings, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return settings, err
		}
		switch key {
		case "display_currency":
			settings.DisplayCurrency = domain.Currency(value)
		case "price_source":
			settings.PriceSource = domain.PriceSource(value)
		case "metalprice_api_key":
			settings.MetalpriceAPIKey = value
		case "middleman_base_url":
			settings.MiddlemanBaseURL = value
		case "auto_lock_minutes":
			minutes, parseErr := strconv.Atoi(value)
			if parseErr == nil && minutes > 0 {
				settings.AutoLockMinutes = minutes
			}
		case "spot_price_unit":
			if value == string(domain.SpotPriceUnitKilogram) ||
				value == string(domain.SpotPriceUnitTroyOz) ||
				value == string(domain.SpotPriceUnitGram) {
				settings.SpotPriceUnit = domain.SpotPriceUnit(value)
			}
		case "ui_theme":
			if value == string(domain.UIThemeLight) || value == string(domain.UIThemeDark) {
				settings.UITheme = domain.UITheme(value)
			}
		}
	}
	return settings, rows.Err()
}

func (database *DB) UpdateSettings(settings domain.AppSettings) error {
	if settings.SpotPriceUnit == "" {
		settings.SpotPriceUnit = domain.SpotPriceUnitTroyOz
	}
	if settings.UITheme == "" {
		settings.UITheme = domain.UIThemeDark
	}
	pairs := map[string]string{
		"display_currency":   string(settings.DisplayCurrency),
		"price_source":       string(settings.PriceSource),
		"metalprice_api_key": settings.MetalpriceAPIKey,
		"middleman_base_url": settings.MiddlemanBaseURL,
		"auto_lock_minutes":  strconv.Itoa(settings.AutoLockMinutes),
		"spot_price_unit":    string(settings.SpotPriceUnit),
		"ui_theme":           string(settings.UITheme),
	}
	tx, err := database.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for key, value := range pairs {
		if _, err := tx.Exec(
			`INSERT INTO settings(key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			key, value,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (database *DB) GetSetting(key string) (string, error) {
	var value string
	err := database.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}
