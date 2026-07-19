package service

import (
	"fmt"
	"time"
)

func resolveDateRange(fromDate string, toDate string, defaultMonths int) (time.Time, time.Time, error) {
	end := time.Now().UTC()
	start := end.AddDate(0, -defaultMonths+1, 0)
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)

	if fromDate != "" {
		parsed, err := time.Parse("2006-01-02", fromDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date")
		}
		start = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	}
	if toDate != "" {
		parsed, err := time.Parse("2006-01-02", toDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date")
		}
		end = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, time.UTC)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end date must be on or after start date")
	}
	return start, end, nil
}
