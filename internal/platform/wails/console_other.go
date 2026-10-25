//go:build !windows

package wailsplatform

// HideChildConsole нужен только на Windows.
func HideChildConsole() {}
