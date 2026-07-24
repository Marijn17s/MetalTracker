//go:build windows

package update

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// startInstaller uses ShellExecute so Windows can show the UAC prompt.
func startInstaller(path string) error {
	operation, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	directory, err := windows.UTF16PtrFromString(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	if err := windows.ShellExecute(0, operation, file, nil, directory, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	return nil
}