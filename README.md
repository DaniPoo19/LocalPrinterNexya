# LocalPrinterNexya - Agente de Impresión Térmica

Agente local de alto rendimiento en **Go** para Windows que conecta aplicaciones web de Punto de Venta (POS) con impresoras térmicas ESC/POS y cajones monedero vía Windows Spooler (`winspool.drv`).

Compatible con **Heladería La Coquera** (`https://gestion.heladerialacoquera.app`) y **Nexya Software** (`https://*.nexya.software`).

---

## 🚀 Compilación y Despliegue

Para compilar el binario para producción en modo GUI (sin ventana de consola negra) y generar el paquete distribuible:

```cmd
build.bat
```

Esto generará la carpeta `LocalPrinter/` con:
- `LocalPrinter\nexya-printer.exe`
- `LocalPrinter\config.json`
- `LocalPrinter\README.md`

---

## 🔒 Arquitectura de Seguridad

1. **Aislamiento en Loopback (`127.0.0.1:18181`):**
   - El socket TCP solo escucha peticiones de la propia máquina física. No se expone a la red local ni a Internet.
   - Es técnicamente imposible que se cruce información con otros clientes o sucursales.
2. **Control Estricto de Orígenes (CORS):**
   - Implementado en `internal/api/middleware.go`.
   - Dominios autorizados: `https://gestion.heladerialacoquera.app`, `*.heladerialacoquera.app`, `*.nexya.software`, `localhost`.
   - Cualquier petición desde dominios externos o no autorizados es rechazada con `403 Forbidden`.
3. **Control Anti-Duplicación y Anti-DoS:**
   - Descarte de órdenes duplicadas enviadas en ventanas menores a 3 segundos.
   - Límite de carga máxima de 2 MB por solicitud (`http.MaxBytesReader`).

---

## 🛠️ Endpoints de la API REST

- `GET /api/health` - Estado del agente y versión activa.
- `GET /api/printers` - Lista de impresoras instaladas en Windows.
- `POST /api/print/order` - Impresión de ticket de orden estructurado.
- `POST /api/print/raster` - Impresión en modo raster/imagen.
- `POST /api/print/raw` - Envío directo de bytes ESC/POS.
- `POST /api/drawer/open` - Apertura directa de gaveta de dinero.
- `POST /api/test` - Impresión de comprobante diagnóstico.
- `GET /api/jobs` - Consulta de historial de trabajos locales.
- `GET /api/config` - Obtener configuración actual.
- `POST /api/config` - Guardar nueva configuración.
