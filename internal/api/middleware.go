package api

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// isOriginAllowed verifica que el origen provenga de dominios autorizados de Nexya, La Coquera o localhost
func isOriginAllowed(origin string) bool {
	if origin == "" {
		// Peticiones de la ventana local de escritorio o herramientas de sistema
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())

	// 1. Permitir localhost y 127.0.0.1 en cualquier puerto (entorno local / pruebas)
	if host == "localhost" || host == "127.0.0.1" {
		return true
	}

	// Los dominios web en la nube deben usar HTTPS
	if u.Scheme != "https" {
		return false
	}

	// 2. Permitir dominios oficiales de Nexya y Heladería La Coquera (HTTPS)
	if host == "gestion.heladerialacoquera.app" ||
		host == "heladerialacoquera.app" ||
		strings.HasSuffix(host, ".heladerialacoquera.app") ||
		host == "gestion.nexya.software" ||
		host == "nexya.software" ||
		strings.HasSuffix(host, ".nexya.software") {
		return true
	}

	// 3. Permitir dominios de staging / preview oficiales
	if strings.HasSuffix(host, ".vercel.app") || strings.HasSuffix(host, ".netlify.app") {
		return true
	}

	return false
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Si la petición viene desde un navegador con cabecera Origin
		if origin != "" {
			if !isOriginAllowed(origin) {
				log.Printf("[SEGURIDAD - BLOQUEADO] Intento de acceso desde sitio web no autorizado: %s (IP: %s, Ruta: %s)", origin, r.RemoteAddr, r.URL.Path)
				http.Error(w, "Acceso no autorizado: Dominio no permitido", http.StatusForbidden)
				return
			}
			// Retornar exactamente el origen validado (nunca wildcard * con credenciales)
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else {
			// Llamadas locales internas directas
			w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:18181")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Access-Control-Request-Private-Network")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Protección DoS: Limitar cuerpo de la petición a máximo 2 MB
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)

		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s - %v", r.Method, r.URL.Path, time.Since(start))
	})
}
