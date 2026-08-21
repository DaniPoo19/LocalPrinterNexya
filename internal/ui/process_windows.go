//go:build windows

package ui

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func isPortActive(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

// KillPreviousInstances busca y finaliza cualquier proceso previo de nexya-printer.exe
// que no sea el proceso actual, liberando los recursos y el puerto TCP.
func KillPreviousInstances(port string) {
	currentPid := os.Getpid()
	if port == "" {
		port = "18181"
	}
	addr := "127.0.0.1:" + port

	// 1. Si el puerto está en uso, solicitar apagado limpio al proceso anterior
	if isPortActive(addr) {
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s/api/shutdown", port), nil)
		if err == nil {
			client := &http.Client{Timeout: 500 * time.Millisecond}
			_, _ = client.Do(req)
		}
	}

	// 2. Esperar a que el proceso anterior se apague voluntariamente
	for i := 0; i < 8; i++ {
		if !isPortActive(addr) {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}

	// 3. Si sigue ocupado, forzar cierre con taskkill excluyendo nuestro PID actual
	cmd := exec.Command("taskkill", "/F", "/FI", "IMAGENAME eq nexya-printer*", "/FI", fmt.Sprintf("PID ne %d", currentPid))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	_ = cmd.Run()

	for i := 0; i < 5; i++ {
		if !isPortActive(addr) {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
}
