//go:build !windows

package hardware

type PrinterInfo struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

func GetDefaultPrinterName() string {
	return "Predefinida"
}

func ListInstalledPrinters() ([]PrinterInfo, error) {
	return []PrinterInfo{
		{Name: "Predefinida", IsDefault: true},
	}, nil
}
