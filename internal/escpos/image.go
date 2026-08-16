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
	"strings"
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

// DownloadAndRasterizeLogo descarga la imagen del logo y la convierte en mapa de bits ESC/POS (GS v 0) con tamaño calibrado en milímetros reales a 203 DPI
func DownloadAndRasterizeLogo(logoURL string, paperWidth string, logoMaxWidthMm int) ([]byte, error) {
	if logoURL == "" {
		return nil, nil
	}

	if logoMaxWidthMm <= 0 {
		logoMaxWidthMm = 25
	}

	cacheKey := fmt.Sprintf("%s:%d:%s", paperWidth, logoMaxWidthMm, logoURL)
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

	// 203 DPI = 8 puntos por milímetro (8 dots/mm)
	// Papel 80mm: Ancho imprimible 72mm = 576 dots (Logo balanceado = 230 dots / 29mm)
	// Papel 58mm: Ancho imprimible 48mm = 384 dots (Logo balanceado = 150 dots / 19mm)
	maxHeadDots := 576
	minLogoDots := 230
	if strings.EqualFold(strings.TrimSpace(paperWidth), "58mm") {
		maxHeadDots = 384
		minLogoDots = 150
	}

	maxDots := logoMaxWidthMm * 8
	if maxDots < minLogoDots {
		maxDots = minLogoDots // Mínimo 29mm para proporción exacta
	}
	if maxDots > maxHeadDots {
		maxDots = maxHeadDots
	}

	rasterBytes := ImageToEscposRaster(img, maxDots)
	if len(rasterBytes) > 0 {
		logoCache.Store(cacheKey, rasterBytes)
	}

	return rasterBytes, nil
}

// ImageToEscposRaster convierte un image.Image en comando ESC/POS GS v 0 0 usando tramado Floyd-Steinberg para máxima fidelidad
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

	// Crear matriz de luminancia 2D en escala de grises [0.0 - 255.0]
	grayMatrix := make([][]float64, targetH)
	for y := 0; y < targetH; y++ {
		grayMatrix[y] = make([]float64, targetW)
		srcY := bounds.Min.Y + (y * origH / targetH)
		for x := 0; x < targetW; x++ {
			srcX := bounds.Min.X + (x * origW / targetW)
			r, g, bl, a := src.At(srcX, srcY).RGBA()

			// Si es transparente (alpha < 50%), fondo blanco (255)
			if a < 128*257 {
				grayMatrix[y][x] = 255.0
			} else {
				// Luminancia estándar perceptiva
				lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257.0
				grayMatrix[y][x] = lum
			}
		}
	}

	// Floyd-Steinberg error diffusion
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			oldPixel := grayMatrix[y][x]
			var newPixel float64
			if oldPixel < 128.0 {
				newPixel = 0.0 // Negro
			} else {
				newPixel = 255.0 // Blanco
			}
			grayMatrix[y][x] = newPixel
			quantError := oldPixel - newPixel

			if x+1 < targetW {
				grayMatrix[y][x+1] += quantError * 7.0 / 16.0
			}
			if y+1 < targetH {
				if x > 0 {
					grayMatrix[y+1][x-1] += quantError * 3.0 / 16.0
				}
				grayMatrix[y+1][x] += quantError * 5.0 / 16.0
				if x+1 < targetW {
					grayMatrix[y+1][x+1] += quantError * 1.0 / 16.0
				}
			}
		}
	}

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
		for xByte := 0; xByte < widthInBytes; xByte++ {
			var b byte = 0
			for bit := 0; bit < 8; bit++ {
				x := xByte*8 + bit
				if x < targetW {
					if grayMatrix[y][x] < 128.0 {
						b |= (1 << (7 - bit)) // Punto negro térmico
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
