//go:build windows

package service

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const localeShortDate = 0x0000001F

var (
	kernel32        = windows.NewLazySystemDLL("kernel32.dll")
	getLocaleInfoEx = kernel32.NewProc("GetLocaleInfoEx")
)

// DateFormatPattern returns the user's configured Windows short-date pattern.
func DateFormatPattern() string {
	buffer := make([]uint16, 80)
	length, _, _ := getLocaleInfoEx.Call(
		0,
		localeShortDate,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if length <= 1 {
		return ""
	}
	return windows.UTF16ToString(buffer[:length-1])
}