//go:build !windows

package ui

func IsAutostartEnabled() bool {
	return false
}

func SetAutostart(enable bool) error {
	return nil
}
