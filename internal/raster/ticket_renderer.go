package raster

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"local-printer-nexya/internal/escpos"
)

var (
	arialFontRegular *opentype.Font
	arialFontBold    *opentype.Font
	fontInitOnce     sync.Once
)

func initFonts() {
	fontInitOnce.Do(func() {
		regBytes, err := os.ReadFile(`C:\Windows\Fonts\arial.ttf`)
		if err == nil {
			arialFontRegular, _ = opentype.Parse(regBytes)
		} else {
			log.Printf("[Raster Font] No se pudo cargar arial.ttf: %v", err)
		}

		boldBytes, err := os.ReadFile(`C:\Windows\Fonts\arialbd.ttf`)
		if err == nil {
			arialFontBold, _ = opentype.Parse(boldBytes)
		} else {
			log.Printf("[Raster Font] No se pudo cargar arialbd.ttf: %v", err)
		}
	})
}

func getFontFace(isBold bool, sizePx float64) font.Face {
	initFonts()
	var f *opentype.Font
	if isBold && arialFontBold != nil {
		f = arialFontBold
	} else if arialFontRegular != nil {
		f = arialFontRegular
	}

	if f != nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size:    sizePx,
			DPI:     72, // 1 pt = 1 px a 72 DPI
			Hinting: font.HintingFull,
		})
		if err == nil {
			return face
		}
	}

	return basicfont.Face7x13
}

type Canvas struct {
	width  int
	height int
	img    *image.RGBA
	curY   int
}

func NewCanvas(width int, initialHeight int) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, width, initialHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	return &Canvas{
		width:  width,
		height: initialHeight,
		img:    img,
		curY:   12,
	}
}

func (c *Canvas) ensureHeight(requiredY int) {
	if requiredY+60 > c.height {
		newH := c.height * 2
		if newH < requiredY+300 {
			newH = requiredY + 300
		}
		newImg := image.NewRGBA(image.Rect(0, 0, c.width, newH))
		draw.Draw(newImg, newImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
		draw.Draw(newImg, c.img.Bounds(), c.img, image.Point{}, draw.Src)
		c.img = newImg
		c.height = newH
	}
}

// DrawImage aplica escalado y tramado Floyd-Steinberg para imprimir logos con nitidez fotográfica
func (c *Canvas) DrawImage(src image.Image, targetWidth int) {
	if src == nil {
		return
	}
	b := src.Bounds()
	origW := b.Dx()
	origH := b.Dy()
	if origW == 0 || origH == 0 {
		return
	}

	targetH := (origH * targetWidth) / origW
	if targetH == 0 {
		targetH = 1
	}

	startX := (c.width - targetWidth) / 2
	c.ensureHeight(c.curY + targetH + 20)

	// 1. Matriz de escala de grises
	grayMatrix := make([][]float64, targetH)
	for y := 0; y < targetH; y++ {
		grayMatrix[y] = make([]float64, targetWidth)
		srcY := b.Min.Y + (y * origH / targetH)
		for x := 0; x < targetWidth; x++ {
			srcX := b.Min.X + (x * origW / targetWidth)
			r, g, bl, a := src.At(srcX, srcY).RGBA()
			if a < 128*257 {
				grayMatrix[y][x] = 255.0 // Fondo transparente = blanco
			} else {
				lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257.0
				grayMatrix[y][x] = lum
			}
		}
	}

	// 2. Floyd-Steinberg error diffusion dithering
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetWidth; x++ {
			oldPixel := grayMatrix[y][x]
			var newPixel float64
			if oldPixel < 140.0 {
				newPixel = 0.0 // Negro
			} else {
				newPixel = 255.0 // Blanco
			}
			grayMatrix[y][x] = newPixel
			quantError := oldPixel - newPixel

			if x+1 < targetWidth {
				grayMatrix[y][x+1] += quantError * 7.0 / 16.0
			}
			if y+1 < targetH {
				if x > 0 {
					grayMatrix[y+1][x-1] += quantError * 3.0 / 16.0
				}
				grayMatrix[y+1][x] += quantError * 5.0 / 16.0
				if x+1 < targetWidth {
					grayMatrix[y+1][x+1] += quantError * 1.0 / 16.0
				}
			}
		}
	}

	// 3. Pintar en el canvas
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetWidth; x++ {
			if grayMatrix[y][x] < 128.0 {
				c.img.Set(startX+x, c.curY+y, color.Black)
			}
		}
	}

	c.curY += targetH + 14
}

func (c *Canvas) DrawText(text string, isBold bool, sizePx float64, align string) {
	face := getFontFace(isBold, sizePx)
	lines := wrapText(text, face, c.width-24)

	for _, line := range lines {
		adv := font.MeasureString(face, line)
		textWidth := adv.Ceil()

		var startX int
		switch align {
		case "center":
			startX = (c.width - textWidth) / 2
		case "right":
			startX = c.width - textWidth - 12
		default: // "left"
			startX = 12
		}

		c.ensureHeight(c.curY + int(sizePx*1.4))
		dot := fixed.P(startX, c.curY+int(sizePx*0.9))

		d := &font.Drawer{
			Dst:  c.img,
			Src:  image.NewUniform(color.Black),
			Face: face,
			Dot:  dot,
		}
		d.DrawString(line)

		c.curY += int(sizePx * 1.3)
	}
}

func (c *Canvas) DrawRow2Cols(leftText string, rightText string, isBold bool, sizePx float64) {
	face := getFontFace(isBold, sizePx)
	rightAdv := font.MeasureString(face, rightText).Ceil()
	maxLeftWidth := c.width - rightAdv - 36

	leftLines := wrapText(leftText, face, maxLeftWidth)
	if len(leftLines) == 0 {
		leftLines = []string{""}
	}

	c.ensureHeight(c.curY + int(sizePx*float64(len(leftLines))*1.4))

	// Dibujar columna derecha alineada arriba
	rightX := c.width - rightAdv - 12
	dRight := &font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(color.Black),
		Face: face,
		Dot:  fixed.P(rightX, c.curY+int(sizePx*0.9)),
	}
	dRight.DrawString(rightText)

	// Dibujar líneas de columna izquierda
	for _, l := range leftLines {
		dLeft := &font.Drawer{
			Dst:  c.img,
			Src:  image.NewUniform(color.Black),
			Face: face,
			Dot:  fixed.P(12, c.curY+int(sizePx*0.9)),
		}
		dLeft.DrawString(l)
		c.curY += int(sizePx * 1.3)
	}

	c.curY += 2
}

func (c *Canvas) DrawDashedLine() {
	c.ensureHeight(c.curY + 16)
	c.curY += 6
	for x := 12; x < c.width-12; x++ {
		if (x/8)%2 == 0 {
			c.img.Set(x, c.curY, color.Black)
			c.img.Set(x, c.curY+1, color.Black)
		}
	}
	c.curY += 10
}

func (c *Canvas) DrawSolidLine(thickness int) {
	c.ensureHeight(c.curY + thickness + 14)
	c.curY += 6
	for y := 0; y < thickness; y++ {
		for x := 12; x < c.width-12; x++ {
			c.img.Set(x, c.curY+y, color.Black)
		}
	}
	c.curY += thickness + 8
}

func (c *Canvas) DrawDoubleLine() {
	c.DrawSolidLine(2)
	c.curY += 2
	c.DrawSolidLine(2)
}

func (c *Canvas) Crop() image.Image {
	return c.img.SubImage(image.Rect(0, 0, c.width, c.curY+20))
}

func wrapText(text string, face font.Face, maxWidth int) []string {
	var result []string
	paragraphs := strings.Split(text, "\n")

	for _, p := range paragraphs {
		words := strings.Fields(p)
		if len(words) == 0 {
			continue
		}

		currentLine := words[0]
		for _, w := range words[1:] {
			testLine := currentLine + " " + w
			if font.MeasureString(face, testLine).Ceil() <= maxWidth {
				currentLine = testLine
			} else {
				result = append(result, currentLine)
				currentLine = w
			}
		}
		if currentLine != "" {
			result = append(result, currentLine)
		}
	}
	return result
}

// RenderOrderTicketToImage dibuja un ticket completo en mapa de bits con tipografía TrueType Arial real y tamaños configurados
func RenderOrderTicketToImage(req *escpos.PrintOrderRequest) image.Image {
	width := 576 // 80mm = 576 dots
	if strings.EqualFold(strings.TrimSpace(req.PaperWidth), "58mm") {
		width = 384 // 58mm = 384 dots
	}

	canvas := NewCanvas(width, 1400)

	// Factor de escala basado en los parámetros de la empresa/sucursal (ej: 95% para Joche)
	scalePercent := req.FontScalePercent
	if scalePercent <= 0 {
		scalePercent = 95
	}
	scaleRatio := float64(scalePercent) / 100.0

	// Tamaños de fuente calibrados a 203 DPI (8 dots/mm)
	fsTitle := 34.0 * scaleRatio       // ~32px Arial Bold (4mm de altura física)
	fsSubtitle := 22.0 * scaleRatio    // ~21px Arial
	fsOrder := 38.0 * scaleRatio       // ~36px Arial Bold
	fsItemName := 28.0 * scaleRatio    // ~26.5px Arial Bold (3.3mm)
	fsBody := 23.0 * scaleRatio        // ~22px Arial
	fsDetail := 20.0 * scaleRatio      // ~19px Arial
	fsTotal := 36.0 * scaleRatio       // ~34px Arial Bold (4.25mm)
	fsChange := 26.0 * scaleRatio      // ~24.5px Arial Bold
	fsFooter := 22.0 * scaleRatio      // ~21px Arial

	// 1. Logo
	if req.ShowLogo && req.LogoUrl != "" {
		logoImg := fetchImage(req.LogoUrl)
		if logoImg != nil {
			logoMaxWidthMm := req.LogoMaxWidth
			if logoMaxWidthMm <= 0 {
				logoMaxWidthMm = 25 // 25mm por defecto
			}
			logoDots := logoMaxWidthMm * 8 // 25mm = 200 dots
			if logoDots < 180 {
				logoDots = 180
			}
			if logoDots > 260 {
				logoDots = 260
			}
			canvas.DrawImage(logoImg, logoDots)
		}
	}

	// 2. Encabezado de Empresa
	bizName := req.BusinessName
	if bizName == "" {
		bizName = "Al carbón de Joche"
	}
	canvas.DrawText(bizName, true, fsTitle, "center")

	if req.NIT != "" {
		canvas.DrawText(fmt.Sprintf("NIT: %s", req.NIT), false, fsSubtitle, "center")
	}
	if req.Address != "" {
		canvas.DrawText(req.Address, false, fsSubtitle, "center")
	}
	if req.Phone != "" {
		canvas.DrawText(fmt.Sprintf("Tel: %s", req.Phone), false, fsSubtitle, "center")
	}

	// 3. Banner de Pedido
	canvas.DrawDashedLine()
	displayCode := req.DailyCode
	if displayCode == "" {
		displayCode = req.OrderCode
	}
	canvas.DrawText(fmt.Sprintf("PEDIDO #%s", displayCode), true, fsOrder, "center")
	canvas.DrawDashedLine()

	// 4. Metadatos de Orden
	if req.CreatedAt != "" {
		canvas.DrawRow2Cols("Fecha:", req.CreatedAt, false, fsBody)
	}
	if req.TableNumber != "" && req.TableNumber != "0" {
		canvas.DrawRow2Cols("Mesa:", req.TableNumber, true, fsBody+2)
	}
	if req.WaiterName != "" {
		canvas.DrawRow2Cols("Mesero:", req.WaiterName, false, fsBody)
	}
	if req.CustomerName != "" {
		canvas.DrawRow2Cols("Cliente:", req.CustomerName, false, fsBody)
	}
	if req.CustomerPhone != "" {
		canvas.DrawRow2Cols("Teléfono:", req.CustomerPhone, false, fsBody)
	}
	if req.DeliveryAddress != "" {
		canvas.DrawRow2Cols("Dirección:", req.DeliveryAddress, false, fsBody)
	}

	// 5. Productos
	canvas.DrawDashedLine()
	canvas.DrawRow2Cols("CANT PRODUCTO", "TOTAL", true, fsBody)
	canvas.DrawDashedLine()

	for _, p := range req.Products {
		left := fmt.Sprintf("%dx %s", p.Quantity, p.Name)
		right := fmt.Sprintf("$ %s", formatCurrency(p.Price*float64(p.Quantity)))
		canvas.DrawRow2Cols(left, right, true, fsItemName)

		for _, cust := range p.Customizations {
			canvas.DrawText(fmt.Sprintf("  • %s", cust), false, fsDetail, "left")
		}
		if p.Observation != "" {
			canvas.DrawText(fmt.Sprintf("  Nota: %s", p.Observation), false, fsDetail, "left")
		}
		canvas.curY += 4
	}

	// 6. Totales
	canvas.DrawDashedLine()
	if req.DeliveryCost > 0 {
		canvas.DrawRow2Cols("Subtotal:", fmt.Sprintf("$ %s", formatCurrency(req.Subtotal)), false, fsBody)
		canvas.DrawRow2Cols("Domicilio:", fmt.Sprintf("$ %s", formatCurrency(req.DeliveryCost)), false, fsBody)
	}
	canvas.DrawDoubleLine()
	canvas.DrawRow2Cols("TOTAL:", fmt.Sprintf("$ %s", formatCurrency(req.Total)), true, fsTotal)
	canvas.DrawDoubleLine()

	// 7. Métodos de Pago
	if req.CashAmount > 0 || strings.EqualFold(req.PaymentType, "CASH") || strings.EqualFold(req.PaymentType, "EFECTIVO") {
		canvas.curY += 6
		if req.CashBillDenomination > 0 {
			canvas.DrawRow2Cols("COBRAR EN EFECTIVO:", fmt.Sprintf("$ %s", formatCurrency(req.CashBillDenomination)), true, fsChange)
			canvas.DrawRow2Cols("VUELTO A DAR:", fmt.Sprintf("$ %s", formatCurrency(req.ChangeAmount)), true, fsChange+2)
		} else {
			canvas.DrawRow2Cols("PAGO EFECTIVO:", fmt.Sprintf("$ %s", formatCurrency(req.Total)), true, fsChange)
		}
		canvas.curY += 6
	} else if req.TransferAmount > 0 {
		canvas.DrawRow2Cols("TRANSFERENCIA:", fmt.Sprintf("$ %s", formatCurrency(req.TransferAmount)), true, fsChange)
	}

	// 8. Pie de Página
	canvas.DrawDashedLine()
	canvas.DrawText("No válido como factura de venta", true, fsFooter-2, "center")
	footer := req.FooterMessage
	if footer == "" {
		footer = "¡Gracias por su compra!\nVuelva pronto"
	}
	for _, l := range strings.Split(footer, "\n") {
		if strings.TrimSpace(l) != "" {
			canvas.DrawText(l, false, fsFooter, "center")
		}
	}

	return canvas.Crop()
}

func fetchImage(url string) image.Image {
	if url == "" {
		return nil
	}
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil
	}
	return img
}

func formatCurrency(val float64) string {
	intVal := int64(val)
	str := fmt.Sprintf("%d", intVal)
	n := len(str)
	if n <= 3 {
		return str
	}
	var res []byte
	rem := n % 3
	if rem > 0 {
		res = append(res, str[:rem]...)
	}
	for i := rem; i < n; i += 3 {
		if len(res) > 0 {
			res = append(res, '.')
		}
		res = append(res, str[i:i+3]...)
	}
	return string(res)
}
