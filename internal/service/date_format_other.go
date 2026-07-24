//go:build !windows

package service

// DateFormatPattern returns an empty pattern so non-Windows platforms use Intl.
func DateFormatPattern() string {
	return ""
}