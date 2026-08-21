//go:build windows

package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const runRegistryKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
const appName = `NexyaLocalPrinter`

// IsAutostartEnabled verifica si el programa está registrado para iniciar con Windows
func IsAutostartEnabled() bool {
	cmd := exec.Command("reg", "query", runRegistryKey, "/v", appName)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), appName)
}

// SetAutostart activa o desactiva el inicio automático con Windows
func SetAutostart(enable bool) error {
	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("no se pudo determinar la ruta del ejecutable: %v", err)
		}
		absPath, err := filepath.Abs(exePath)
		if err != nil {
			absPath = exePath
		}
		cmd := exec.Command("reg", "add", runRegistryKey, "/v", appName, "/t", "REG_SZ", "/d", absPath, "/f")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error al registrar en Windows: %v (%s)", err, strings.TrimSpace(string(out)))
		}
		return nil
	} else {
		cmd := exec.Command("reg", "delete", runRegistryKey, "/v", appName, "/f")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		_ = cmd.Run()
		return nil
	}
}
