//go:build !windows

package hardware

import "fmt"

func PrintRawToPrinter(printerName string, data []byte) error {
	return fmt.Errorf("impresión directa con winspool solo está disponible en Windows")
}
