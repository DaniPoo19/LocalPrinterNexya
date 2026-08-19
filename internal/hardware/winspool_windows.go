//go:build windows

package hardware

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	winspool             = syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter      = winspool.NewProc("OpenPrinterW")
	procClosePrinter     = winspool.NewProc("ClosePrinter")
	procStartDocPrinter  = winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter    = winspool.NewProc("EndDocPrinter")
	procStartPagePrinter = winspool.NewProc("StartPagePrinter")
	procEndPagePrinter   = winspool.NewProc("EndPagePrinter")
	procWritePrinter     = winspool.NewProc("WritePrinter")
)

type DOC_INFO_1 struct {
	pDocName    *uint16
	pOutputFile *uint16
	pDatatype   *uint16
}

func isVirtualPrinter(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "pdf") ||
		strings.Contains(lower, "onenote") ||
		strings.Contains(lower, "xps") ||
		strings.Contains(lower, "fax") ||
		strings.Contains(lower, "send to") ||
		lower == "predefinida" ||
		lower == ""
}

// PrintRawToPrinter envía la secuencia binaria directa al Spooler de Windows
func PrintRawToPrinter(printerName string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("no hay datos para imprimir")
	}

	// 1. Resolver el nombre real de la impresora física
	targetPrinter := strings.TrimSpace(printerName)
	if targetPrinter == "" || targetPrinter == "Predefinida" || strings.EqualFold(targetPrinter, "default") {
		targetPrinter = GetDefaultPrinterName()
	}

	// Si la impresora resultante es virtual (PDF/OneNote/Fax) o inválida, buscar una impresora térmica POS física
	if isVirtualPrinter(targetPrinter) {
		printers, _ := ListInstalledPrinters()
		var posPrinter string
		for _, p := range printers {
			if isVirtualPrinter(p.Name) {
				continue
			}
			lower := strings.ToLower(p.Name)
			if strings.Contains(lower, "pos") || strings.Contains(lower, "thermal") ||
				strings.Contains(lower, "receipt") || strings.Contains(lower, "ticket") ||
				strings.Contains(lower, "58") || strings.Contains(lower, "80") ||
				strings.Contains(lower, "xp-") || strings.Contains(lower, "tm-") ||
				strings.Contains(lower, "star") || strings.Contains(lower, "bixolon") {
				posPrinter = p.Name
				break
			}
			if posPrinter == "" {
				posPrinter = p.Name
			}
		}
		if posPrinter != "" {
			targetPrinter = posPrinter
		}
	}

	if targetPrinter == "" || targetPrinter == "Predefinida" {
		return fmt.Errorf("no se encontró ninguna impresora física POS instalada en Windows")
	}

	pNamePtr, err := syscall.UTF16PtrFromString(targetPrinter)
	if err != nil {
		return fmt.Errorf("nombre de impresora inválido '%s': %v", targetPrinter, err)
	}

	var hPrinter uintptr
	ret, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pNamePtr)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("no se pudo abrir la impresora Windows '%s' (código de error: %v)", targetPrinter, err)
	}
	defer procClosePrinter.Call(hPrinter)

	docName, _ := syscall.UTF16PtrFromString("Documento POS Nexya")
	dataType, _ := syscall.UTF16PtrFromString("RAW")

	docInfo := DOC_INFO_1{
		pDocName:    docName,
		pOutputFile: nil,
		pDatatype:   dataType,
	}

	ret, _, err = procStartDocPrinter.Call(
		hPrinter,
		1,
		uintptr(unsafe.Pointer(&docInfo)),
	)
	if ret == 0 {
		return fmt.Errorf("error al iniciar el documento en el Spooler de Windows para '%s': %v", targetPrinter, err)
	}
	defer procEndDocPrinter.Call(hPrinter)

	procStartPagePrinter.Call(hPrinter)

	var bytesWritten uint32
	ret, _, err = procWritePrinter.Call(
		hPrinter,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&bytesWritten)),
	)

	procEndPagePrinter.Call(hPrinter)

	if ret == 0 || bytesWritten != uint32(len(data)) {
		return fmt.Errorf("error escribiendo datos a la impresora '%s' (escritos %d de %d bytes): %v", targetPrinter, bytesWritten, len(data), err)
	}

	return nil
}
