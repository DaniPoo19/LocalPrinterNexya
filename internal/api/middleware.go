package api

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// isOriginAllowed verifica que el origen provenga de dominios autorizados de Nexya o de localhost
func isOriginAllowed(origin string) bool {
	if origin == "" {
		// Peticiones de la ventana local de escritorio o herramientas de sistema
		return true
	}

	// Normalizar minúsculas
	originLower := strings.ToLower(origin)

	// 1. Permitir localhost y 127.0.0.1 en cualquier puerto (entorno local / pruebas)
	if strings.HasPrefix(originLower, "http://localhost") ||
		strings.HasPrefix(originLower, "http://127.0.0.1") ||
		strings.HasPrefix(originLower, "https://localhost") ||
		strings.HasPrefix(originLower, "https://127.0.0.1") {
		return true
	}

	// 2. Permitir dominios oficiales de Nexya (HTTPS)
	if strings.HasPrefix(originLower, "https://gestion.nexya.software") ||
		strings.HasPrefix(originLower, "https://nexya.software") ||
		strings.HasSuffix(originLower, ".nexya.software") {
		return true
	}

	// 3. Permitir dominios de staging / preview oficiales
	if strings.HasSuffix(originLower, ".vercel.app") || strings.HasSuffix(originLower, ".netlify.app") {
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
