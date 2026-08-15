package escpos

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

var (
	logoCache = sync.Map{}
	httpClient = &http.Client{
		Timeout: 6 * time.Second,
	}
)

// DownloadAndRasterizeLogo descarga la imagen del logo y la convierte en mapa de bits ESC/POS (GS v 0) con tamaño proporcional y caché
func DownloadAndRasterizeLogo(logoURL string, paperWidth string, logoMaxWidthPercent int) ([]byte, error) {
	if logoURL == "" {
		return nil, nil
	}

	if logoMaxWidthPercent <= 0 {
		logoMaxWidthPercent = 25
	}

	cacheKey := fmt.Sprintf("%s:%d:%s", paperWidth, logoMaxWidthPercent, logoURL)
	if cached, ok := logoCache.Load(cacheKey); ok {
		if data, ok := cached.([]byte); ok && len(data) > 0 {
			return data, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", logoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	// Base total dots: 384 para 80mm, 256 para 58mm
	baseDots := 384
	if paperWidth == "58mm" {
		baseDots = 256
	}

	// Calcular ancho proporcional según el porcentaje de Mongo (ej: 25% de 384 = 96 dots)
	maxDots := int(float64(baseDots) * (float64(logoMaxWidthPercent) / 100.0))
	if maxDots < 48 {
		maxDots = 48
	}
	if maxDots > baseDots {
		maxDots = baseDots
	}

	rasterBytes := ImageToEscposRaster(img, maxDots)
	if len(rasterBytes) > 0 {
		logoCache.Store(cacheKey, rasterBytes)
	}

	return rasterBytes, nil
}

// ImageToEscposRaster convierte un image.Image en comando ESC/POS GS v 0 0
func ImageToEscposRaster(src image.Image, maxWidth int) []byte {
	bounds := src.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	if origW <= 0 || origH <= 0 {
		return nil
	}

	targetW := origW
	targetH := origH

	if targetW > maxWidth {
		targetH = int(float64(origH) * float64(maxWidth) / float64(origW))
		targetW = maxWidth
	}

	// Alinear ancho a múltiplo de 8
	widthInBytes := (targetW + 7) / 8
	targetW = widthInBytes * 8

	var buf bytes.Buffer

	// Centrar imagen
	buf.Write(CmdAlignCenter)

	// GS v 0 0 xL xH yL yH
	xL := byte(widthInBytes % 256)
	xH := byte(widthInBytes / 256)
	yL := byte(targetH % 256)
	yH := byte(targetH / 256)

	buf.Write([]byte{0x1D, 0x76, 0x30, 0x00, xL, xH, yL, yH})

	for y := 0; y < targetH; y++ {
		srcY := bounds.Min.Y + (y * origH / targetH)
		for xByte := 0; xByte < widthInBytes; xByte++ {
			var b byte = 0
			for bit := 0; bit < 8; bit++ {
				x := xByte*8 + bit
				if x < targetW {
					srcX := bounds.Min.X + (x * origW / targetW)
					r, g, bl, a := src.At(srcX, srcY).RGBA()

					// Si es transparente, fondo blanco
					if a < 128*257 {
						// Blanco (bit 0)
					} else {
						// Luminancia estándar (0.299 R + 0.587 G + 0.114 B)
						lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257.0
						if lum < 185.0 {
							b |= (1 << (7 - bit)) // Punto negro
						}
					}
				}
			}
			buf.WriteByte(b)
		}
	}

	// Restaurar alineación y salto de línea
	buf.Write(CmdLineFeed)
	buf.Write(CmdAlignLeft)

	return buf.Bytes()
}
