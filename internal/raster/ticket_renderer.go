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
		curY:   10,
	}
}

func (c *Canvas) ensureHeight(requiredY int) {
	if requiredY+40 > c.height {
		newH := c.height * 2
		if newH < requiredY+200 {
			newH = requiredY + 200
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
	srcW := b.Dx()
	srcH := b.Dy()
	if srcW == 0 || srcH == 0 {
		return
	}

	targetH := (srcH * targetWidth) / srcW
	startX := (c.width - targetWidth) / 2
	c.ensureHeight(c.curY + targetH)

	for y := 0; y < targetH; y++ {
		srcY := b.Min.Y + (y * srcH / targetH)
		for x := 0; x < targetWidth; x++ {
			srcX := b.Min.X + (x * srcW / targetWidth)
			r, g, bl, a := src.At(srcX, srcY).RGBA()
			if a > 128*257 {
				lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257.0
				if lum < 150.0 {
					c.img.Set(startX+x, c.curY+y, color.Black)
				}
			}
		}
	}

	c.curY += targetH + 10
}

func (c *Canvas) DrawText(text string, isBold bool, sizePx float64, align string) {
	face := getFontFace(isBold, sizePx)
	lines := wrapText(text, face, c.width-20)

	for _, line := range lines {
		adv := font.MeasureString(face, line)
		textWidth := adv.Ceil()

		var startX int
		switch align {
		case "center":
			startX = (c.width - textWidth) / 2
		case "right":
			startX = c.width - textWidth - 10
		default: // "left"
			startX = 10
		}

		c.ensureHeight(c.curY + int(sizePx*1.3))
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

	c.ensureHeight(c.curY + int(sizePx*float64(len(leftLines))*1.3))

	// Dibujar columna derecha alineada arriba
	rightX := c.width - rightAdv - 10
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
			Dot:  fixed.P(10, c.curY+int(sizePx*0.9)),
		}
		dLeft.DrawString(l)
		c.curY += int(sizePx * 1.25)
	}

	c.curY += 2
}

func (c *Canvas) DrawDashedLine() {
	c.ensureHeight(c.curY + 12)
	c.curY += 4
	for x := 10; x < c.width-10; x++ {
		if (x/6)%2 == 0 {
			c.img.Set(x, c.curY, color.Black)
			c.img.Set(x, c.curY+1, color.Black)
		}
	}
	c.curY += 8
}

func (c *Canvas) DrawSolidLine(thickness int) {
	c.ensureHeight(c.curY + thickness + 10)
	c.curY += 4
	for y := 0; y < thickness; y++ {
		for x := 10; x < c.width-10; x++ {
			c.img.Set(x, c.curY+y, color.Black)
		}
	}
	c.curY += thickness + 6
}

func (c *Canvas) DrawDoubleLine() {
	c.DrawSolidLine(2)
	c.curY += 1
	c.DrawSolidLine(2)
}

func (c *Canvas) Crop() image.Image {
	return c.img.SubImage(image.Rect(0, 0, c.width, c.curY+15))
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

// RenderOrderTicketToImage dibuja un ticket completo en mapa de bits con tipografía TrueType Arial real
func RenderOrderTicketToImage(req *escpos.PrintOrderRequest) image.Image {
	width := 576 // 80mm = 576 dots
	if strings.EqualFold(strings.TrimSpace(req.PaperWidth), "58mm") {
		width = 384 // 58mm = 384 dots
	}

	canvas := NewCanvas(width, 1200)

	// 1. Logo
	if req.ShowLogo && req.LogoUrl != "" {
		logoImg := fetchImage(req.LogoUrl)
		if logoImg != nil {
			logoDots := 230
			if width == 384 {
				logoDots = 150
			}
			if req.LogoMaxWidth > 0 {
				calcDots := req.LogoMaxWidth * 8
				if calcDots < logoDots {
					logoDots = calcDots
				}
			}
			canvas.DrawImage(logoImg, logoDots)
		}
	}

	// 2. Encabezado de Empresa
	bizName := req.BusinessName
	if bizName == "" {
		bizName = "Al carbón de Joche"
	}
	canvas.DrawText(bizName, true, 26, "center")

	if req.NIT != "" {
		canvas.DrawText(fmt.Sprintf("NIT: %s", req.NIT), false, 17, "center")
	}
	if req.Address != "" {
		canvas.DrawText(req.Address, false, 17, "center")
	}
	if req.Phone != "" {
		canvas.DrawText(fmt.Sprintf("Tel: %s", req.Phone), false, 17, "center")
	}

	// 3. Banner de Pedido
	canvas.DrawDashedLine()
	displayCode := req.DailyCode
	if displayCode == "" {
		displayCode = req.OrderCode
	}
	canvas.DrawText(fmt.Sprintf("PEDIDO #%s", displayCode), true, 28, "center")
	canvas.DrawDashedLine()

	// 4. Metadatos de Orden
	if req.CreatedAt != "" {
		canvas.DrawRow2Cols("Fecha:", req.CreatedAt, false, 17)
	}
	if req.TableNumber != "" && req.TableNumber != "0" {
		canvas.DrawRow2Cols("Mesa:", req.TableNumber, true, 19)
	}
	if req.WaiterName != "" {
		canvas.DrawRow2Cols("Mesero:", req.WaiterName, false, 17)
	}
	if req.CustomerName != "" {
		canvas.DrawRow2Cols("Cliente:", req.CustomerName, false, 17)
	}
	if req.CustomerPhone != "" {
		canvas.DrawRow2Cols("Teléfono:", req.CustomerPhone, false, 17)
	}
	if req.DeliveryAddress != "" {
		canvas.DrawRow2Cols("Dirección:", req.DeliveryAddress, false, 17)
	}

	// 5. Productos
	canvas.DrawDashedLine()
	canvas.DrawRow2Cols("CANT PRODUCTO", "TOTAL", true, 18)
	canvas.DrawDashedLine()

	for _, p := range req.Products {
		left := fmt.Sprintf("%dx %s", p.Quantity, p.Name)
		right := fmt.Sprintf("$ %s", formatCurrency(p.Price*float64(p.Quantity)))
		canvas.DrawRow2Cols(left, right, true, 20)

		for _, cust := range p.Customizations {
			canvas.DrawText(fmt.Sprintf("  • %s", cust), false, 15, "left")
		}
		if p.Observation != "" {
			canvas.DrawText(fmt.Sprintf("  Nota: %s", p.Observation), false, 15, "left")
		}
		canvas.curY += 3
	}

	// 6. Totales
	canvas.DrawDashedLine()
	if req.DeliveryCost > 0 {
		canvas.DrawRow2Cols("Subtotal:", fmt.Sprintf("$ %s", formatCurrency(req.Subtotal)), false, 18)
		canvas.DrawRow2Cols("Domicilio:", fmt.Sprintf("$ %s", formatCurrency(req.DeliveryCost)), false, 18)
	}
	canvas.DrawDoubleLine()
	canvas.DrawRow2Cols("TOTAL:", fmt.Sprintf("$ %s", formatCurrency(req.Total)), true, 28)
	canvas.DrawDoubleLine()

	// 7. Métodos de Pago
	if req.CashAmount > 0 || strings.EqualFold(req.PaymentType, "CASH") || strings.EqualFold(req.PaymentType, "EFECTIVO") {
		canvas.curY += 4
		if req.CashBillDenomination > 0 {
			canvas.DrawRow2Cols("COBRAR EN EFECTIVO:", fmt.Sprintf("$ %s", formatCurrency(req.CashBillDenomination)), true, 19)
			canvas.DrawRow2Cols("VUELTO A DAR:", fmt.Sprintf("$ %s", formatCurrency(req.ChangeAmount)), true, 21)
		} else {
			canvas.DrawRow2Cols("PAGO EFECTIVO:", fmt.Sprintf("$ %s", formatCurrency(req.Total)), true, 19)
		}
		canvas.curY += 4
	} else if req.TransferAmount > 0 {
		canvas.DrawRow2Cols("TRANSFERENCIA:", fmt.Sprintf("$ %s", formatCurrency(req.TransferAmount)), true, 19)
	}

	// 8. Pie de Página
	canvas.DrawDashedLine()
	canvas.DrawText("No válido como factura de venta", true, 16, "center")
	footer := req.FooterMessage
	if footer == "" {
		footer = "¡Gracias por su compra!\nVuelva pronto"
	}
	for _, l := range strings.Split(footer, "\n") {
		if strings.TrimSpace(l) != "" {
			canvas.DrawText(l, false, 17, "center")
		}
	}

	return canvas.Crop()
}

func fetchImage(url string) image.Image {
	if url == "" {
		return nil
	}
	client := &http.Client{Timeout: 5 * time.Second}
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
