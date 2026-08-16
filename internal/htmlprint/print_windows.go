//go:build windows

package htmlprint

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"local-printer-nexya/internal/hardware"
)

var browserPaths = []string{
	`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
}

// PrintHtmlSilently imprime un documento HTML en segundo plano utilizando el motor Chromium Headless de Windows
func PrintHtmlSilently(printerName string, htmlContent string, copies int) error {
	if strings.TrimSpace(htmlContent) == "" {
		return fmt.Errorf("el contenido HTML está vacío")
	}

	targetPrinter := strings.TrimSpace(printerName)
	if targetPrinter == "" || targetPrinter == "Predefinida" || strings.EqualFold(targetPrinter, "default") {
		targetPrinter = hardware.GetDefaultPrinterName()
	}

	if targetPrinter == "" || targetPrinter == "Predefinida" {
		printers, _ := hardware.ListInstalledPrinters()
		if len(printers) > 0 {
			targetPrinter = printers[0].Name
		}
	}

	if targetPrinter == "" || targetPrinter == "Predefinida" {
		return fmt.Errorf("no se encontró ninguna impresora instalada para impresión silenciosa")
	}

	// 1. Guardar el HTML en un archivo temporal
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("nexya_ticket_%d.html", time.Now().UnixNano()))
	if err := os.WriteFile(tempFile, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("error guardando HTML temporal: %v", err)
	}
	defer func() {
		// Borrar después de un breve delay para que el proceso del navegador termine de leerlo
		go func() {
			time.Sleep(10 * time.Second)
			_ = os.Remove(tempFile)
		}()
	}()

	// 2. Localizar el ejecutable de Edge o Chrome
	var browserExe string
	for _, p := range browserPaths {
		if _, err := os.Stat(p); err == nil {
			browserExe = p
			break
		}
	}

	if browserExe == "" {
		return fmt.Errorf("no se encontró Microsoft Edge ni Google Chrome en el sistema")
	}

	if copies < 1 {
		copies = 1
	}

	// 3. Ejecutar comando headless con --print-to-printer
	for i := 0; i < copies; i++ {
		cmd := exec.Command(browserExe,
			"--headless",
			"--disable-gpu",
			"--no-pdf-header-footer",
			fmt.Sprintf("--print-to-printer=%s", targetPrinter),
			tempFile,
		)

		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("[HTMLPrint ERROR] Fallo al imprimir copia %d/%d con '%s': %v | Output: %s", i+1, copies, browserExe, err, string(output))
			return fmt.Errorf("fallo al imprimir con motor Chromium: %v", err)
		}

		log.Printf("[HTMLPrint OK] Copia %d/%d enviada a '%s' vía Chromium Headless", i+1, copies, targetPrinter)
		if copies > 1 && i < copies-1 {
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}
