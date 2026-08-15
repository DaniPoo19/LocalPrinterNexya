//go:build windows

package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var browserAppPaths = []string{
	`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
}

// OpenAppWindow abre la interfaz gráfica en modo ventana de escritorio nativa y monitorea el cierre
func OpenAppWindow(url string) {
	tempProfile := filepath.Join(os.TempDir(), "nexya_printer_app_profile")

	for _, browserPath := range browserAppPaths {
		if _, err := os.Stat(browserPath); err == nil {
			cmd := exec.Command(browserPath, "--app="+url, "--window-size=940,740", "--user-data-dir="+tempProfile)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				HideWindow: true,
			}
			if err := cmd.Start(); err == nil {
				go func() {
					_ = cmd.Wait()
					os.Exit(0)
				}()
				return
			}
		}
	}

	// Fallback al navegador predeterminado del sistema (Sin consola visible)
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	_ = cmd.Start()
}
