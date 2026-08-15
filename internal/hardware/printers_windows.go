//go:build windows

package hardware

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	procGetDefaultPrinter = winspool.NewProc("GetDefaultPrinterW")
	procEnumPrinters      = winspool.NewProc("EnumPrintersW")
)

type PrinterInfo struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

type PRINTER_INFO_4 struct {
	PPrinterName *uint16
	PServerName  *uint16
	Attributes   uint32
}

// GetDefaultPrinterName obtiene el nombre de la impresora predeterminada de Windows directamente en memoria
func GetDefaultPrinterName() string {
	var bufSize uint32 = 256
	buf := make([]uint16, bufSize)

	ret, _, _ := procGetDefaultPrinter.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufSize)),
	)
	if ret != 0 {
		return syscall.UTF16ToString(buf)
	}
	return "Predefinida"
}

// ListInstalledPrinters obtiene la lista de impresoras mediante la API nativa EnumPrintersW (Sin abrir ventanas CMD ni PowerShell)
func ListInstalledPrinters() ([]PrinterInfo, error) {
	defaultPrinter := GetDefaultPrinterName()
	var result []PrinterInfo

	flags := uint32(0x00000002 | 0x00000004) // PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS
	level := uint32(4)                        // PRINTER_INFO_4W (Ligero y rápido)

	var bytesNeeded, numPrinters uint32

	// Primera llamada para determinar el tamaño del buffer necesario
	procEnumPrinters.Call(
		uintptr(flags),
		0,
		uintptr(level),
		0,
		0,
		uintptr(unsafe.Pointer(&bytesNeeded)),
		uintptr(unsafe.Pointer(&numPrinters)),
	)

	if bytesNeeded > 0 {
		buf := make([]byte, bytesNeeded)
		ret, _, _ := procEnumPrinters.Call(
			uintptr(flags),
			0,
			uintptr(level),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(bytesNeeded),
			uintptr(unsafe.Pointer(&bytesNeeded)),
			uintptr(unsafe.Pointer(&numPrinters)),
		)

		if ret != 0 && numPrinters > 0 {
			sliceHeader := (*[1024]PRINTER_INFO_4)(unsafe.Pointer(&buf[0]))
			for i := 0; i < int(numPrinters); i++ {
				pInfo := sliceHeader[i]
				if pInfo.PPrinterName != nil {
					name := syscall.UTF16ToString((*[512]uint16)(unsafe.Pointer(pInfo.PPrinterName))[:])
					if strings.TrimSpace(name) != "" {
						result = append(result, PrinterInfo{
							Name:      name,
							IsDefault: strings.EqualFold(name, defaultPrinter),
						})
					}
				}
			}
		}
	}

	// Si no se listaron impresoras pero hay predeterminada, devolverla
	if len(result) == 0 && defaultPrinter != "" && defaultPrinter != "Predefinida" {
		result = append(result, PrinterInfo{
			Name:      defaultPrinter,
			IsDefault: true,
		})
	}

	return result, nil
}
