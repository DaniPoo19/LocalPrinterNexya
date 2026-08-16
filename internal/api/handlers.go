package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"local-printer-nexya/internal/config"
	"local-printer-nexya/internal/escpos"
	"local-printer-nexya/internal/hardware"
	"local-printer-nexya/internal/raster"
	"local-printer-nexya/internal/ui"
)

var startTime = time.Now()

type Server struct {
	cfg *config.Config
}

func NewServer(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	printers, _ := hardware.ListInstalledPrinters()
	def := hardware.GetDefaultPrinterName()

	resp := HealthResponse{
		Status:            "online",
		Version:           "1.0.0",
		DefaultPrinter:    def,
		InstalledPrinters: printers,
		Config:            config.GetConfig(),
		UptimeSeconds:     int64(time.Since(startTime).Seconds()),
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) HandleGetPrinters(w http.ResponseWriter, r *http.Request) {
	printers, err := hardware.ListInstalledPrinters()
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error obteniendo impresoras: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"printers": printers,
		"default":  hardware.GetDefaultPrinterName(),
	})
}

func (s *Server) HandlePrintOrder(w http.ResponseWriter, r *http.Request) {
	var req escpos.PrintOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Payload inválido: %v", err),
		})
		return
	}

	cfg := config.GetConfig()

	// 1. La impresora se define por la app local si fue seleccionada
	if cfg.DefaultPrinter != "" && cfg.DefaultPrinter != "Predefinida" {
		req.PrinterName = cfg.DefaultPrinter
	} else if req.PrinterName == "" || req.PrinterName == "Predefinida" {
		req.PrinterName = cfg.DefaultPrinter
	}

	// 2. El ancho de papel físico siempre lo impone la configuración local de la máquina
	if cfg.PaperWidth != "" {
		req.PaperWidth = cfg.PaperWidth
	} else if req.PaperWidth == "" {
		req.PaperWidth = "80mm"
	}

	// 3. Corte y apertura de gaveta
	req.CutPaper = cfg.AutoCut
	if cfg.OpenDrawer && (req.CashAmount > 0 || req.SaleType == "COUNTER_SALE" || req.OpenDrawer) {
		req.OpenDrawer = true
	}

	// 4. Copias físicas controladas por la aplicación local
	copies := cfg.DefaultCopies
	if copies < 1 {
		copies = 1
	}
	if copies > 10 {
		copies = 10
	}

	// 5. Renderizar el ticket con tipografía TrueType Arial real (203 DPI) a mapa de bits ESC/POS
	ticketImg := raster.RenderOrderTicketToImage(&req)

	maxWidth := 576
	if strings.EqualFold(strings.TrimSpace(req.PaperWidth), "58mm") {
		maxWidth = 384
	}

	b := escpos.NewBuilder(req.PaperWidth)
	if req.Beep {
		b.Beep()
	}
	if req.OpenDrawer {
		b.OpenDrawer()
	}

	rasterBytes := escpos.ImageToEscposRaster(ticketImg, maxWidth)
	b.PrintRasterImage(rasterBytes)

	if req.CutPaper {
		b.Cut(true) // Avance de 7 líneas completo + corte
	} else {
		b.FeedLines(7)
	}

	payload := b.Bytes()

	var lastErr error
	for i := 0; i < copies; i++ {
		err := hardware.PrintRawToPrinter(req.PrinterName, payload)
		if err != nil {
			lastErr = err
			log.Printf("[PrintOrder ERROR] Error enviando copia %d/%d a '%s': %v", i+1, copies, req.PrinterName, err)
			break
		}
		if copies > 1 && i < copies-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	// Registrar en el historial de trabajos
	jobRecord := PrintJobRecord{
		OrderCode:   req.OrderCode,
		PrinterName: req.PrinterName,
		Copies:      copies,
		Total:       req.Total,
		Success:     lastErr == nil,
		RawPayload:  payload,
	}
	AddJobRecord(jobRecord)

	if lastErr != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error enviando ticket a la impresora '%s': %v", req.PrinterName, lastErr),
		})
		return
	}

	log.Printf("[PrintOrder OK] Pedido #%s impreso exitosamente (%d copia/s) en '%s' con tipografía Arial TrueType", req.OrderCode, copies, req.PrinterName)
	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Pedido #%s impreso exitosamente (%d copia/s)", req.OrderCode, copies),
	})
}

func (s *Server) HandlePrintRaw(w http.ResponseWriter, r *http.Request) {
	var req PrintRawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   "Payload JSON inválido",
		})
		return
	}

	cfg := config.GetConfig()
	printer := req.PrinterName
	if printer == "" {
		printer = cfg.DefaultPrinter
	}

	var data []byte
	if len(req.RawBytes) > 0 {
		data = req.RawBytes
	} else if req.TextContent != "" {
		b := escpos.NewBuilder(cfg.PaperWidth)
		if req.OpenDrawer {
			b.OpenDrawer()
		}
		b.PrintLine(req.TextContent)
		if req.CutPaper {
			b.Cut(true)
		}
		data = b.Bytes()
	} else {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   "Se requiere 'raw_bytes' o 'text_content'",
		})
		return
	}

	err := hardware.PrintRawToPrinter(printer, data)

	AddJobRecord(PrintJobRecord{
		OrderCode:   "RAW_DATA",
		PrinterName: printer,
		Copies:      1,
		Success:     err == nil,
		RawPayload:  data,
	})

	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error enviando datos RAW: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Datos RAW impresos correctamente",
	})
}

func (s *Server) HandlePrintRaster(w http.ResponseWriter, r *http.Request) {
	var req PrintRasterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   "Payload JSON inválido",
		})
		return
	}

	if req.ImageBase64 == "" {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   "El campo 'image_base64' es requerido",
		})
		return
	}

	cfg := config.GetConfig()
	printer := req.PrinterName
	if printer == "" || printer == "Predefinida" {
		printer = cfg.DefaultPrinter
	}

	paperWidth := req.PaperWidth
	if paperWidth == "" {
		paperWidth = cfg.PaperWidth
	}
	if paperWidth == "" {
		paperWidth = "80mm"
	}

	// 1. Limpiar prefijo data URL si existe
	base64Data := req.ImageBase64
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error decodificando imagen base64: %v", err),
		})
		return
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error leyendo formato de imagen gráfica: %v", err),
		})
		return
	}

	maxWidth := 576
	if strings.EqualFold(strings.TrimSpace(paperWidth), "58mm") {
		maxWidth = 384
	}

	b := escpos.NewBuilder(paperWidth)
	if req.Beep {
		b.Beep()
	}
	if req.OpenDrawer || cfg.OpenDrawer {
		b.OpenDrawer()
	}

	rasterBytes := escpos.ImageToEscposRaster(img, maxWidth)
	b.PrintRasterImage(rasterBytes)

	if req.CutPaper || cfg.AutoCut {
		b.Cut(true)
	} else {
		b.FeedLines(7)
	}

	payload := b.Bytes()
	copies := req.Copies
	if copies < 1 {
		copies = cfg.DefaultCopies
	}
	if copies < 1 {
		copies = 1
	}

	var lastErr error
	for i := 0; i < copies; i++ {
		err := hardware.PrintRawToPrinter(printer, payload)
		if err != nil {
			lastErr = err
			log.Printf("[PrintRaster ERROR] Error enviando imagen a '%s': %v", printer, err)
			break
		}
		if copies > 1 && i < copies-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	orderCode := req.OrderCode
	if orderCode == "" {
		orderCode = "RASTER_GRAPHIC"
	}

	AddJobRecord(PrintJobRecord{
		OrderCode:   orderCode,
		PrinterName: printer,
		Copies:      copies,
		Success:     lastErr == nil,
		RawPayload:  payload,
	})

	if lastErr != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error imprimiendo ticket gráfico en '%s': %v", printer, lastErr),
		})
		return
	}

	log.Printf("[PrintRaster OK] Ticket gráfico 100%% Arial impreso (%d copia/s) en '%s'", copies, printer)
	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Ticket gráfico Arial impreso exitosamente (%d copia/s)", copies),
	})
}

func (s *Server) HandleOpenDrawer(w http.ResponseWriter, r *http.Request) {
	var req OpenDrawerRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	cfg := config.GetConfig()
	printer := req.PrinterName
	if printer == "" {
		printer = cfg.DefaultPrinter
	}

	b := escpos.NewBuilder(cfg.PaperWidth)
	b.OpenDrawer()

	err := hardware.PrintRawToPrinter(printer, b.Bytes())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error abriendo cajón monedero: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Pulso de apertura de cajón enviado",
	})
}

func (s *Server) HandleTestPrint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PrinterName string `json:"printer_name,omitempty"`
		Copies      int    `json:"copies,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	cfg := config.GetConfig()
	printer := cfg.DefaultPrinter
	if printer == "" || printer == "Predefinida" {
		if req.PrinterName != "" && req.PrinterName != "Predefinida" {
			printer = req.PrinterName
		} else {
			printer = cfg.DefaultPrinter
		}
	}

	copies := cfg.DefaultCopies
	if copies < 1 {
		copies = 1
	}
	if copies > 5 {
		copies = 5
	}

	paperWidth := cfg.PaperWidth
	if paperWidth == "" {
		paperWidth = "80mm"
	}

	payload := escpos.FormatTestTicket(paperWidth)
	var lastErr error
	for i := 0; i < copies; i++ {
		err := hardware.PrintRawToPrinter(printer, payload)
		if err != nil {
			lastErr = err
			break
		}
		if copies > 1 && i < copies-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	AddJobRecord(PrintJobRecord{
		OrderCode:   "TEST_DIAG",
		PrinterName: printer,
		Copies:      copies,
		Success:     lastErr == nil,
		RawPayload:  payload,
	})

	if lastErr != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error imprimiendo prueba: %v", lastErr),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Ticket de prueba impreso (%d copia/s)", copies),
	})
}

func (s *Server) HandleGetJobs(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"jobs":    GetJobRecords(),
	})
}

func (s *Server) HandleClearJobs(w http.ResponseWriter, r *http.Request) {
	ClearJobRecords()
	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Historial de trabajos limpiado",
	})
}

func (s *Server) HandleReprintJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.JobID == "" {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   "Se requiere 'job_id'",
		})
		return
	}

	job := GetJobRecordByID(req.JobID)
	if job == nil || len(job.RawPayload) == 0 {
		s.writeJSON(w, http.StatusNotFound, ApiResponse{
			Success: false,
			Error:   "Trabajo no encontrado en el buffer de memoria",
		})
		return
	}

	cfg := config.GetConfig()
	printer := job.PrinterName
	if printer == "" || printer == "Predefinida" {
		printer = cfg.DefaultPrinter
	}

	err := hardware.PrintRawToPrinter(printer, job.RawPayload)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error al re-imprimir: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Trabajo #%s re-impreso exitosamente", job.OrderCode),
	})
}

func (s *Server) HandleGetAutostart(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"enabled": ui.IsAutostartEnabled(),
	})
}

func (s *Server) HandleSetAutostart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   "Payload JSON inválido",
		})
		return
	}

	err := ui.SetAutostart(req.Enabled)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error configurando inicio con Windows: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Configuración de inicio con Windows actualizada",
	})
}

func (s *Server) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, config.GetConfig())
}

func (s *Server) HandleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		s.writeJSON(w, http.StatusBadRequest, ApiResponse{
			Success: false,
			Error:   "Configuración inválida",
		})
		return
	}

	if err := config.SaveConfig(&newCfg); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   fmt.Sprintf("Error guardando config: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Configuración actualizada",
	})
}

func (s *Server) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Servicio cerrándose...",
	})
	go func() {
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()
}
