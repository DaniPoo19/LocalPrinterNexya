package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		origin   string
		expected bool
	}{
		// Orígenes permitidos locales
		{"", true},
		{"http://localhost:5173", true},
		{"http://127.0.0.1:18181", true},
		{"https://localhost:3000", true},
		{"https://127.0.0.1:8080", true},

		// Orígenes oficiales Heladería La Coquera
		{"https://gestion.heladerialacoquera.app", true},
		{"https://heladerialacoquera.app", true},
		{"https://home.heladerialacoquera.app", true},
		{"https://api.heladerialacoquera.app", true},

		// Orígenes oficiales Nexya
		{"https://gestion.nexya.software", true},
		{"https://nexya.software", true},
		{"https://pos.nexya.software", true},

		// Orígenes staging permitidos
		{"https://heladeria-admin.vercel.app", true},
		{"https://heladeria-pos.netlify.app", true},

		// Orígenes NO permitidos (deben ser bloqueados)
		{"https://google.com", false},
		{"https://malicious-site.com", false},
		{"https://otraempresa.app", false},
		{"https://heladerialacoquera.app.fake.com", false},
		{"http://gestion.heladerialacoquera.app.attacker.com", false},
		{"https://nexya.software.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			got := isOriginAllowed(tt.origin)
			if got != tt.expected {
				t.Errorf("isOriginAllowed(%q) = %v; want %v", tt.origin, got, tt.expected)
			}
		})
	}
}

func TestCorsMiddleware_AllowedOrigin(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := CorsMiddleware(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://gestion.heladerialacoquera.app")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Esperaba status 200 OK, obtuvo %d", rec.Code)
	}

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://gestion.heladerialacoquera.app" {
		t.Errorf("Access-Control-Allow-Origin incorrecto: %q", allowOrigin)
	}

	allowPNA := rec.Header().Get("Access-Control-Allow-Private-Network")
	if allowPNA != "true" {
		t.Errorf("Access-Control-Allow-Private-Network esperado 'true', obtuvo: %q", allowPNA)
	}
}

func TestCorsMiddleware_BlockedOrigin(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CorsMiddleware(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://sitio-malicioso.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("Esperaba status 403 Forbidden para sitio no autorizado, obtuvo %d", rec.Code)
	}
}

func TestCorsMiddleware_OptionsPreflight(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // No debería ejecutarse en OPTIONS
	})

	handler := CorsMiddleware(dummyHandler)

	req := httptest.NewRequest(http.MethodOptions, "/api/print/order", nil)
	req.Header.Set("Origin", "https://gestion.heladerialacoquera.app")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Preflight OPTIONS esperaba 200 OK, obtuvo %d", rec.Code)
	}
}
