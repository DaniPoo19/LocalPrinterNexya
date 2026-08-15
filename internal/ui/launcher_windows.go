//go:build windows

package ui

import (
	"os"
	"os/exec"
	"syscall"
)

var browserAppPaths = []string{
	`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
}

// OpenAppWindow abre la interfaz gráfica en modo ventana de escritorio nativa
func OpenAppWindow(url string) {
	for _, browserPath := range browserAppPaths {
		if _, err := os.Stat(browserPath); err == nil {
			cmd := exec.Command(browserPath, "--app="+url, "--window-size=940,740")
			cmd.SysProcAttr = &syscall.SysProcAttr{
				HideWindow: true,
			}
			if err := cmd.Start(); err == nil {
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
