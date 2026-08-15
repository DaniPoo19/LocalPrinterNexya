package api

import (
	"local-printer-nexya/internal/config"
	"local-printer-nexya/internal/hardware"
)

type HealthResponse struct {
	Status            string                  `json:"status"`
	Version           string                  `json:"version"`
	DefaultPrinter    string                  `json:"default_printer"`
	InstalledPrinters []hardware.PrinterInfo `json:"installed_printers"`
	Config            *config.Config          `json:"config"`
	UptimeSeconds     int64                   `json:"uptime_seconds"`
}

type PrintRawRequest struct {
	PrinterName string `json:"printer_name,omitempty"`
	RawBytes    []byte `json:"raw_bytes,omitempty"`
	TextContent string `json:"text_content,omitempty"`
	OpenDrawer  bool   `json:"open_drawer"`
	CutPaper    bool   `json:"cut_paper"`
}

type OpenDrawerRequest struct {
	PrinterName string `json:"printer_name,omitempty"`
}

type ApiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
