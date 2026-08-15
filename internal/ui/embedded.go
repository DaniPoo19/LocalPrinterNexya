package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:web
var webAssets embed.FS

// GetFileSystem retorna el sistema de archivos embebido de la UI
func GetFileSystem() (http.FileSystem, error) {
	sub, err := fs.Sub(webAssets, "web")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// SPAHandler maneja las solicitudes estáticas de la interfaz gráfica
func SPAHandler(staticFS http.FileSystem) http.Handler {
	fileServer := http.FileServer(staticFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// No interceptar peticiones de la API
		if strings.HasPrefix(path, "/api") {
			return
		}

		// Comprobar si existe el archivo solicitado
		f, err := staticFS.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// Redirigir a index.html
			r.URL.Path = "/"
		} else {
			_ = f.Close()
		}

		fileServer.ServeHTTP(w, r)
	})
}
