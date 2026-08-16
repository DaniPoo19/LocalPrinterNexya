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

var saleTypeLabels = map[string]string{
	"DELIVERY":     "Domicilio",
	"PICKUP":       "Recoger en tienda",
	"ON_SITE":      "En mesa",
	"COUNTER_SALE": "Venta en mostrador",
}

var paymentTypeLabels = map[string]string{
	"CASH":          "Efectivo",
	"BANK_TRANSFER": "Transferencia",
	"HYBRID":        "Híbrido (Transf. + Efectivo)",
	"PENDING":       "Pendiente",
	"CREDIT":        "Crédito",
}

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
			DPI:     72,
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

	grayMatrix := make([][]float64, targetH)
	for y := 0; y < targetH; y++ {
		grayMatrix[y] = make([]float64, targetWidth)
		srcY := b.Min.Y + (y * origH / targetH)
		for x := 0; x < targetWidth; x++ {
			srcX := b.Min.X + (x * origW / targetWidth)
			r, g, bl, a := src.At(srcX, srcY).RGBA()
			if a < 128*257 {
				grayMatrix[y][x] = 255.0
			} else {
				lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257.0
				grayMatrix[y][x] = lum
			}
		}
	}

	// Floyd-Steinberg error diffusion
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetWidth; x++ {
			oldPixel := grayMatrix[y][x]
			var newPixel float64
			if oldPixel < 140.0 {
				newPixel = 0.0
			} else {
				newPixel = 255.0
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

		c.curY += int(sizePx * 1.25)
	}
}

func (c *Canvas) DrawRow2Cols(leftText string, rightText string, isBold bool, sizePx float64) {
	face := getFontFace(isBold, sizePx)
	rightAdv := font.MeasureString(face, rightText).Ceil()
	maxLeftWidth := c.width - rightAdv - 30

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
		c.curY += int(sizePx * 1.25)
	}

	c.curY += 2
}

func (c *Canvas) DrawDashedLine() {
	c.ensureHeight(c.curY + 20)
	c.curY += 6
	for x := 12; x < c.width-12; x++ {
		if (x%14) < 8 {
			c.img.Set(x, c.curY, color.Black)
			c.img.Set(x, c.curY+1, color.Black)
		}
	}
	c.curY += 12
}

func (c *Canvas) DrawDottedLine() {
	c.ensureHeight(c.curY + 18)
	c.curY += 6
	for x := 12; x < c.width-12; x++ {
		if (x%8) < 3 {
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

// RenderOrderTicketToImage dibuja un ticket completo en mapa de bits con tipografía TrueType Arial real y estructura idéntica a BusinessAdmin
func RenderOrderTicketToImage(req *escpos.PrintOrderRequest) image.Image {
	width := 576 // 80mm = 576 dots
	if strings.EqualFold(strings.TrimSpace(req.PaperWidth), "58mm") {
		width = 384 // 58mm = 384 dots
	}

	canvas := NewCanvas(width, 1600)

	// Factor de escala exacto CSS (96 DPI) a Dots Térmicos (203 DPI)
	scalePercent := req.FontScalePercent
	if scalePercent <= 0 {
		scalePercent = 95
	}
	scaleRatio := float64(scalePercent) / 100.0
	dpiRatio := 203.0 / 96.0 // 2.114583

	// Tamaños de fuente calibrados 1:1 con el diálogo de Google Chrome
	fsTitle := 28.0 * dpiRatio * scaleRatio       // ~56px Arial Bold (7.0mm)
	fsSubtitle := 18.0 * dpiRatio * scaleRatio    // ~36px Arial (4.5mm)
	fsOrder := 30.0 * dpiRatio * scaleRatio       // ~60px Arial Bold (7.5mm)
	fsItemName := 21.0 * dpiRatio * scaleRatio    // ~42px Arial Bold (5.3mm)
	fsBody := 19.0 * dpiRatio * scaleRatio        // ~38px Arial (4.8mm)
	fsDetail := 16.0 * dpiRatio * scaleRatio      // ~32px Arial (4.0mm)
	fsTotal := 28.0 * dpiRatio * scaleRatio       // ~56px Arial Bold (7.0mm)
	fsChange := 22.0 * dpiRatio * scaleRatio      // ~44px Arial Bold (5.5mm)
	fsFooter := 18.0 * dpiRatio * scaleRatio      // ~36px Arial (4.5mm)

	// 1. LOGO (Condicionado por ShowLogo)
	if req.ShowLogo && req.LogoUrl != "" {
		logoImg := fetchImage(req.LogoUrl)
		if logoImg != nil {
			logoMaxWidthMm := req.LogoMaxWidth
			if logoMaxWidthMm <= 0 {
				logoMaxWidthMm = 25
			}
			logoDots := logoMaxWidthMm * 8
			if logoDots < 180 {
				logoDots = 180
			}
			if logoDots > 260 {
				logoDots = 260
			}
			canvas.DrawImage(logoImg, logoDots)
		}
	}

	// 2. ENCABEZADO DE EMPRESA
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

	canvas.DrawDashedLine()

	// 3. INFO PEDIDO
	displayCode := req.DailyCode
	if displayCode == "" {
		displayCode = req.OrderCode
	}
	canvas.DrawText(fmt.Sprintf("Pedido #%s", displayCode), true, fsOrder, "center")

	if req.CreatedAt != "" {
		canvas.DrawRow2Cols("Fecha:", req.CreatedAt, false, fsBody)
	}

	saleTypeLabel := saleTypeLabels[req.SaleType]
	if saleTypeLabel == "" {
		saleTypeLabel = req.SaleType
	}
	if saleTypeLabel != "" {
		canvas.DrawRow2Cols("Tipo:", saleTypeLabel, false, fsBody)
	}

	paymentLabel := paymentTypeLabels[req.PaymentType]
	if paymentLabel == "" {
		paymentLabel = req.PaymentType
	}
	if paymentLabel != "" {
		canvas.DrawRow2Cols("Pago:", paymentLabel, false, fsBody)
	}

	if req.TableNumber != "" && req.TableNumber != "0" {
		canvas.DrawDottedLine()
		canvas.DrawRow2Cols("Mesa:", req.TableNumber, true, fsBody)
		canvas.DrawDottedLine()
	}

	if req.ShowWaiter && req.WaiterName != "" {
		canvas.DrawRow2Cols("Atendido por:", req.WaiterName, false, fsBody)
	}

	canvas.DrawDashedLine()

	// 4. CLIENTE (Condicionado por ShowCustomer)
	if req.ShowCustomer && req.CustomerName != "" {
		canvas.DrawText(fmt.Sprintf("CLIENTE: %s", req.CustomerName), true, fsItemName, "left")
		if req.CustomerDoc != "" {
			docType := req.CustomerDocType
			if docType == "" {
				docType = "CC"
			}
			canvas.DrawText(fmt.Sprintf("Doc: %s %s", docType, req.CustomerDoc), false, fsDetail, "left")
		}
		if req.CustomerPhone != "" {
			canvas.DrawText(fmt.Sprintf("Tel: %s", req.CustomerPhone), false, fsDetail, "left")
		}
		if req.DeliveryAddress != "" {
			canvas.DrawText(fmt.Sprintf("Dir: %s", req.DeliveryAddress), false, fsDetail, "left")
		}
		canvas.DrawDashedLine()
	}

	// 5. PRODUCTOS
	canvas.DrawText("PRODUCTOS", true, fsBody, "left")

	for _, p := range req.Products {
		takeawayTag := ""
		if p.Takeaway {
			takeawayTag = " (Llevar)"
		}
		nameLeft := fmt.Sprintf("%dx %s%s", p.Quantity, p.Name, takeawayTag)

		unitPrice := p.UnitBasePrice
		if unitPrice <= 0 {
			unitPrice = p.Price
		}
		priceRight := fmt.Sprintf("$ %s", formatCurrency(unitPrice))

		canvas.DrawRow2Cols(nameLeft, priceRight, true, fsItemName)

		if p.VariationName != "" && p.VariationName != "Único" {
			canvas.DrawText(fmt.Sprintf("Variación: %s", p.VariationName), false, fsDetail, "left")
		}
		if p.Size != "" {
			canvas.DrawText(p.Size, false, fsDetail, "left")
		}
		for _, cust := range p.Customizations {
			canvas.DrawText(cust, false, fsDetail, "left")
		}
		if len(p.Addons) > 0 {
			canvas.DrawText("Adiciones:", true, fsDetail, "left")
			for _, add := range p.Addons {
				addLeft := fmt.Sprintf("+ %s", add.Name)
				addRight := ""
				if add.Price > 0 {
					addRight = fmt.Sprintf("$ %s", formatCurrency(add.Price))
				}
				canvas.DrawRow2Cols(addLeft, addRight, false, fsDetail)
			}
		}
		if p.Observation != "" {
			canvas.DrawText(fmt.Sprintf("Nota: %s", p.Observation), false, fsDetail, "left")
		}

		// Fila de Total por Producto alineada a la derecha
		totalProd := p.TotalPrice
		if totalProd <= 0 {
			totalProd = p.Price * float64(p.Quantity)
		}
		if p.Quantity > 1 {
			calcText := fmt.Sprintf("%d × $ %s = $ %s", p.Quantity, formatCurrency(unitPrice), formatCurrency(totalProd))
			canvas.DrawText(calcText, true, fsBody, "right")
		} else {
			canvas.DrawText(fmt.Sprintf("$ %s", formatCurrency(totalProd)), true, fsBody, "right")
		}

		// Separación punteada entre productos tal como en BusinessAdmin
		canvas.DrawDottedLine()
	}

	canvas.DrawDashedLine()

	// 6. SUBTOTAL Y DOMICILIO
	if req.DeliveryCost > 0 {
		canvas.DrawRow2Cols("Subtotal:", fmt.Sprintf("$ %s", formatCurrency(req.Subtotal)), false, fsBody)
		canvas.DrawRow2Cols("Domicilio:", fmt.Sprintf("$ %s", formatCurrency(req.DeliveryCost)), false, fsBody)
		canvas.DrawDashedLine()
	}

	// 7. TOTAL
	canvas.DrawDoubleLine()
	canvas.DrawRow2Cols("TOTAL:", fmt.Sprintf("$ %s", formatCurrency(req.Total)), true, fsTotal)
	canvas.DrawDoubleLine()

	// 8. DETALLE DEL PAGO Y CAJA DE EFECTIVO (Condicionado por ShowPaymentDetails)
	if req.ShowPaymentDetails {
		hasCashOrTransfer := req.CashAmount > 0 || req.TransferAmount > 0 || req.CashBillDenomination > 0 || req.ChangeAmount > 0
		if hasCashOrTransfer {
			canvas.DrawDashedLine()
			canvas.DrawText("DETALLE DEL PAGO", true, fsBody, "left")

			if req.TransferAmount > 0 {
				canvas.DrawRow2Cols("Transferencia:", fmt.Sprintf("$ %s", formatCurrency(req.TransferAmount)), false, fsBody)
			}
			if req.CashAmount > 0 && req.TransferAmount > 0 {
				canvas.DrawRow2Cols("Efectivo:", fmt.Sprintf("$ %s", formatCurrency(req.CashAmount)), false, fsBody)
			}

			// Caja de cobro en efectivo destacada
			if req.CashBillDenomination > 0 || req.ChangeAmount > 0 {
				canvas.curY += 4
				canvas.DrawSolidLine(2)
				canvas.DrawText("COBRAR EN EFECTIVO", true, fsBody, "center")
				canvas.curY += 2

				if req.CashBillDenomination > 0 {
					canvas.DrawRow2Cols("Paga con:", fmt.Sprintf("$ %s", formatCurrency(req.CashBillDenomination)), true, fsBody)
				}
				if req.ChangeAmount > 0 {
					canvas.DrawRow2Cols("VUELTO A DAR:", fmt.Sprintf("$ %s", formatCurrency(req.ChangeAmount)), true, fsChange)
				}
				canvas.DrawSolidLine(2)
				canvas.curY += 4
			}
		}
	}

	// 9. PIE DE PÁGINA
	canvas.DrawDashedLine()
	canvas.DrawText("No válido como factura de venta", true, fsFooter, "center")

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
