package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"local-printer-nexya/internal/api"
	"local-printer-nexya/internal/config"
	"local-printer-nexya/internal/escpos"
	"local-printer-nexya/internal/hardware"
	"local-printer-nexya/internal/ui"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n[ERROR CRÍTICO]: %v\n", r)
			time.Sleep(3 * time.Second)
		}
	}()

	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	port := cfg.Port
	if port == "" {
		port = "18181"
	}
	addr := "127.0.0.1:" + port
	url := "http://" + addr

	// 1. Si el puerto ya está en uso, abrir directamente la ventana de la UI existente
	if isPortInUse(addr) {
		ui.OpenAppWindow(url)
		time.Sleep(1 * time.Second)
		return
	}

	srv := api.NewServer(cfg)
	mux := http.NewServeMux()

	// Endpoints REST
	mux.HandleFunc("/api/health", srv.HandleHealth)
	mux.HandleFunc("/api/printers", srv.HandleGetPrinters)
	mux.HandleFunc("/api/print/order", srv.HandlePrintOrder)
	mux.HandleFunc("/api/print/raster", srv.HandlePrintRaster)
	mux.HandleFunc("/api/print/raw", srv.HandlePrintRaw)
	mux.HandleFunc("/api/drawer/open", srv.HandleOpenDrawer)
	mux.HandleFunc("/api/test", srv.HandleTestPrint)
	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			srv.HandleClearJobs(w, r)
		} else {
			srv.HandleGetJobs(w, r)
		}
	})
	mux.HandleFunc("/api/jobs/reprint", srv.HandleReprintJob)
	mux.HandleFunc("/api/autostart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			srv.HandleSetAutostart(w, r)
		} else {
			srv.HandleGetAutostart(w, r)
		}
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			srv.HandleSaveConfig(w, r)
		} else {
			srv.HandleGetConfig(w, r)
		}
	})
	mux.HandleFunc("/api/shutdown", srv.HandleShutdown)

	// Servir UI Embebida
	staticFS, err := ui.GetFileSystem()
	if err == nil {
		mux.Handle("/", ui.SPAHandler(staticFS))
	}

	handler := api.CorsMiddleware(mux)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Abrir automáticamente la ventana de la aplicación de escritorio
	go func() {
		time.Sleep(300 * time.Millisecond)
		ui.OpenAppWindow(url)
	}()

	// Manejo de comandos interactivos por si se ejecuta en terminal
	defPrinter := hardware.GetDefaultPrinterName()
	go handleConsoleInput(cfg, defPrinter, url)

	// Manejo de señales de apagado del sistema
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case <-serverErr:
		time.Sleep(2 * time.Second)
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}
}

func isPortInUse(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

func handleConsoleInput(cfg *config.Config, defaultPrinter string, appUrl string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(strings.ToLower(input))
		switch cmd {
		case "t", "test":
			payload := escpos.FormatTestTicket(cfg.PaperWidth)
			_ = hardware.PrintRawToPrinter(defaultPrinter, payload)
		case "c", "cajon", "drawer":
			b := escpos.NewBuilder(cfg.PaperWidth)
			b.OpenDrawer()
			_ = hardware.PrintRawToPrinter(defaultPrinter, b.Bytes())
		case "w", "window", "ui":
			ui.OpenAppWindow(appUrl)
		case "q", "exit", "quit":
			os.Exit(0)
		}
	}
}
