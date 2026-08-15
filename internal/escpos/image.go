package escpos

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 2 * time.Second,
}

// DownloadAndRasterizeLogo descarga la imagen del logo y la convierte en mapa de bits ESC/POS (GS v 0)
func DownloadAndRasterizeLogo(logoURL string, paperWidth string) ([]byte, error) {
	if logoURL == "" {
		return nil, nil
	}

	resp, err := httpClient.Get(logoURL)
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

	// Ancho máximo en puntos: 384 para 80mm, 256 para 58mm
	maxDots := 384
	if paperWidth == "58mm" {
		maxDots = 256
	}

	return ImageToEscposRaster(img, maxDots), nil
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
