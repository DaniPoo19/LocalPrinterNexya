//go:build !windows

package htmlprint

import "fmt"

func PrintHtmlSilently(printerName string, htmlContent string, copies int) error {
	return fmt.Errorf("impresión HTML silenciosa solo disponible en Windows")
}
