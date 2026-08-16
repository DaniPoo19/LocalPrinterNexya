package escpos

import (
	"fmt"
	"strings"
	"time"
)

type TicketProduct struct {
	Name           string   `json:"name"`
	Quantity       int      `json:"quantity"`
	Price          float64  `json:"price"`
	Observation    string   `json:"observation,omitempty"`
	Customizations []string `json:"customizations,omitempty"`
}

type PrintOrderRequest struct {
	PrinterName          string          `json:"printer_name,omitempty"`
	PaperWidth           string          `json:"paper_width,omitempty"` // "58mm" or "80mm"
	BusinessName         string          `json:"business_name,omitempty"`
	LogoUrl              string          `json:"logo_url,omitempty"`
	ShowLogo             bool            `json:"show_logo"`
	LogoMaxWidth         int             `json:"logo_max_width,omitempty"`
	FontScalePercent     int             `json:"font_scale_percent,omitempty"`
	NIT                  string          `json:"nit,omitempty"`
	Address              string          `json:"address,omitempty"`
	Phone                string          `json:"phone,omitempty"`
	OrderCode            string          `json:"order_code"`
	DailyCode            string          `json:"daily_code,omitempty"`
	CreatedAt            string          `json:"created_at,omitempty"`
	SaleType             string          `json:"sale_type,omitempty"` // "ON_SITE", "DELIVERY", "PICKUP", "COUNTER_SALE"
	TableNumber          string          `json:"table_number,omitempty"`
	WaiterName           string          `json:"waiter_name,omitempty"`
	CustomerName         string          `json:"customer_name,omitempty"`
	CustomerDoc          string          `json:"customer_doc,omitempty"`
	CustomerDocType      string          `json:"customer_doc_type,omitempty"`
	CustomerPhone        string          `json:"customer_phone,omitempty"`
	DeliveryAddress      string          `json:"delivery_address,omitempty"`
	Products             []TicketProduct `json:"products"`
	Subtotal             float64         `json:"subtotal"`
	DeliveryCost         float64         `json:"delivery_cost,omitempty"`
	Discount             float64         `json:"discount,omitempty"`
	Total                float64         `json:"total"`
	PaymentType          string          `json:"payment_type,omitempty"`
	CashAmount           float64         `json:"cash_amount,omitempty"`
	TransferAmount       float64         `json:"transfer_amount,omitempty"`
	CashBillDenomination float64         `json:"cash_bill_denomination,omitempty"`
	ChangeAmount         float64         `json:"change_amount,omitempty"`
	FooterMessage        string          `json:"footer_message,omitempty"`
	OpenDrawer           bool            `json:"open_drawer"`
	CutPaper             bool            `json:"cut_paper"`
	Beep                 bool            `json:"beep"`
	Copies               int             `json:"copies,omitempty"`
}

func FormatOrderTicket(req *PrintOrderRequest) []byte {
	paperWidth := req.PaperWidth
	if paperWidth == "" {
		paperWidth = "80mm"
	}

	b := NewBuilder(paperWidth)

	if req.Beep {
		b.Beep()
	}

	if req.OpenDrawer {
		b.OpenDrawer()
	}

	// 0. LOGO DE LA EMPRESA (Si está activado)
	if req.ShowLogo && req.LogoUrl != "" {
		logoWidthMm := req.LogoMaxWidth
		if logoWidthMm <= 0 {
			logoWidthMm = 25
		}
		if logoBytes, err := DownloadAndRasterizeLogo(req.LogoUrl, paperWidth, logoWidthMm); err == nil && len(logoBytes) > 0 {
			b.PrintRasterImage(logoBytes)
			b.FeedLines(1)
		}
	}

	// 1. ENCABEZADO EMPRESA
	b.SetAlign("center")
	if req.BusinessName != "" {
		b.SetBold(true).SetDoubleHeight(true)
		b.PrintLine(req.BusinessName)
		b.SetBold(false).SetDoubleHeight(false)
	}

	if req.NIT != "" {
		b.PrintLine(fmt.Sprintf("NIT: %s", req.NIT))
	}
	if req.Address != "" {
		b.PrintLine(req.Address)
	}
	if req.Phone != "" {
		b.PrintLine(fmt.Sprintf("Tel: %s", req.Phone))
	}

	b.PrintDivider("-")

	// 2. DETALLES DEL PEDIDO (BANNER DESTACADO)
	b.SetAlign("center")
	b.SetBold(true).SetDoubleSize(true)
	displayCode := req.OrderCode
	if req.DailyCode != "" {
		displayCode = req.DailyCode
	}
	b.PrintLine(fmt.Sprintf("Pedido #%s", displayCode))
	b.SetBold(false).SetDoubleSize(false)

	b.SetAlign("left")
	dateStr := req.CreatedAt
	if dateStr == "" {
		dateStr = time.Now().Format("02/01/2006 15:04")
	}
	b.PrintRow2Cols("Fecha:", dateStr)

	saleTypeLabel := "En Local"
	if strings.EqualFold(req.SaleType, "DELIVERY") {
		saleTypeLabel = "Domicilio"
	} else if strings.EqualFold(req.SaleType, "PICKUP") {
		saleTypeLabel = "Para Recoger"
	} else if strings.EqualFold(req.SaleType, "COUNTER_SALE") {
		saleTypeLabel = "Venta Mostrador"
	}
	b.PrintRow2Cols("Tipo:", saleTypeLabel)

	if req.PaymentType != "" {
		b.PrintRow2Cols("Pago:", formatPaymentTypeName(req.PaymentType))
	}

	if req.TableNumber != "" && req.TableNumber != "0" {
		b.SetBold(true)
		b.PrintRow2Cols("Mesa:", fmt.Sprintf("Mesa #%s", req.TableNumber))
		b.SetBold(false)
	}

	if req.WaiterName != "" {
		b.PrintRow2Cols("Atendido por:", req.WaiterName)
	}

	// 3. DATOS DEL CLIENTE
	if req.CustomerName != "" {
		b.PrintDivider("-")
		b.SetBold(true).PrintLine(fmt.Sprintf("CLIENTE: %s", req.CustomerName))
		b.SetBold(false)
		if req.CustomerDoc != "" {
			docType := req.CustomerDocType
			if docType == "" {
				docType = "CC"
			}
			b.PrintLine(fmt.Sprintf(" Doc: %s %s", docType, req.CustomerDoc))
		}
		if req.CustomerPhone != "" {
			b.PrintLine(fmt.Sprintf(" Tel: %s", req.CustomerPhone))
		}
		if req.DeliveryAddress != "" {
			b.PrintLine(fmt.Sprintf(" Dir: %s", req.DeliveryAddress))
		}
	}

	b.PrintDivider("-")

	// 4. PRODUCTOS
	b.SetBold(true)
	b.PrintLine("PRODUCTOS")
	b.SetBold(false)
	b.PrintDivider("-")

	for i, p := range req.Products {
		// Cabecera del producto en doble altura negrita (3.5mm de alto, igual a Chrome)
		b.PrintItemRowDoubleHeight(p.Quantity, p.Name, formatCurrency(p.Price))

		// Personalizaciones / Sabores / Adiciones con precio
		for _, cust := range p.Customizations {
			b.PrintLine(fmt.Sprintf("  • %s", cust))
		}

		// Observaciones
		if p.Observation != "" {
			b.PrintLine(fmt.Sprintf("  * NOTA: %s", p.Observation))
		}

		// Subtotal si cantidad > 1
		if p.Quantity > 1 {
			subtotalLine := fmt.Sprintf("%d × %s = %s", p.Quantity, formatCurrency(p.Price), formatCurrency(p.Price*float64(p.Quantity)))
			b.SetBold(true)
			b.PrintRow2Cols("", subtotalLine)
			b.SetBold(false)
		}

		if i < len(req.Products)-1 {
			b.PrintDivider(".")
		}
	}

	b.PrintDivider("-")

	// 5. TOTALES
	if req.Subtotal > 0 && (req.DeliveryCost > 0 || req.Discount > 0) {
		b.PrintRow2Cols("Subtotal:", formatCurrency(req.Subtotal))
	}
	if req.DeliveryCost > 0 {
		b.PrintRow2Cols("Domicilio:", formatCurrency(req.DeliveryCost))
	}
	if req.Discount > 0 {
		b.PrintRow2Cols("Descuento:", fmt.Sprintf("-%s", formatCurrency(req.Discount)))
	}

	b.PrintDivider("=")
	b.PrintDoubleSizeRow2Cols("TOTAL:", formatCurrency(req.Total))
	b.PrintDivider("=")

	// 6. DETALLES DE PAGO Y VUELTO (COBRAR EN EFECTIVO)
	hasPaymentBreakdown := req.CashAmount > 0 || req.TransferAmount > 0 || req.CashBillDenomination > 0 || req.ChangeAmount > 0
	if hasPaymentBreakdown {
		b.SetBold(true).PrintLine("DETALLE DEL PAGO")
		b.SetBold(false)

		if req.TransferAmount > 0 {
			b.PrintRow2Cols("Transferencia:", formatCurrency(req.TransferAmount))
		}
		if req.CashAmount > 0 && req.TransferAmount > 0 {
			b.PrintRow2Cols("Efectivo:", formatCurrency(req.CashAmount))
		}

		if req.CashBillDenomination > 0 || req.ChangeAmount > 0 {
			b.PrintDivider("-")
			b.SetAlign("center")
			b.SetBold(true).SetDoubleHeight(true).PrintLine("COBRAR EN EFECTIVO")
			b.SetBold(false).SetDoubleHeight(false).SetAlign("left")
			if req.CashBillDenomination > 0 {
				b.PrintRow2Cols("Paga con:", formatCurrency(req.CashBillDenomination))
			}
			if req.ChangeAmount > 0 {
				b.SetBold(true).SetDoubleHeight(true)
				b.PrintRow2Cols("VUELTO A DAR:", formatCurrency(req.ChangeAmount))
				b.SetBold(false).SetDoubleHeight(false)
			}
		}
	}

	// 7. PIE DE PÁGINA
	b.PrintDivider("-")
	b.SetAlign("center")
	b.SetBold(true).PrintLine("No válido como factura de venta")
	b.SetBold(false)

	footer := req.FooterMessage
	if footer == "" {
		footer = "¡Gracias por su compra!\nVuelva pronto"
	}
	for _, line := range strings.Split(footer, "\n") {
		if strings.TrimSpace(line) != "" {
			b.PrintLine(line)
		}
	}

	// 8. CORTE
	if req.CutPaper {
		b.Cut(true)
	} else {
		b.FeedLines(4)
	}

	return b.Bytes()
}

func FormatTestTicket(paperWidth string) []byte {
	if paperWidth == "" {
		paperWidth = "80mm"
	}
	b := NewBuilder(paperWidth)
	b.SetAlign("center")
	b.SetBold(true).SetDoubleSize(true)
	b.PrintLine("NEXYA PRINTER")
	b.SetBold(false).SetDoubleSize(false)
	b.PrintLine("Impresión de Diagnóstico")
	b.PrintDivider("=")
	b.SetAlign("left")
	b.PrintRow2Cols("Estado:", "OK - Conectado")
	b.PrintRow2Cols("Ancho:", paperWidth)
	b.PrintRow2Cols("Fecha:", time.Now().Format("02/01/2006 15:04:05"))
	b.PrintDivider("-")
	b.SetAlign("center")
	b.PrintLine("Hardware Spooler: Operativo")
	b.PrintLine("Corte de papel: Activo")
	b.PrintDivider("=")
	b.SetBold(true).PrintLine("No válido como factura de venta")
	b.SetBold(false)
	b.PrintLine("Sistema Listo para Operar")
	b.Cut(true)
	return b.Bytes()
}

func formatCurrency(val float64) string {
	intVal := int64(val)
	str := fmt.Sprintf("%d", intVal)
	var parts []string
	for len(str) > 3 {
		parts = append([]string{str[len(str)-3:]}, parts...)
		str = str[:len(str)-3]
	}
	if len(str) > 0 {
		parts = append([]string{str}, parts...)
	}
	return fmt.Sprintf("$ %s", strings.Join(parts, "."))
}

func formatPaymentTypeName(pt string) string {
	switch strings.ToUpper(strings.TrimSpace(pt)) {
	case "CASH", "EFECTIVO":
		return "Efectivo"
	case "TRANSFER", "TRANSFERENCIA":
		return "Transferencia Bancaria"
	case "HYBRID", "HIBRIDO", "HÍBRIDO":
		return "Híbrido (Transf. + Efectivo)"
	case "CARD", "TARJETA", "DATAFONO":
		return "Datáfono / Tarjeta"
	case "CREDIT", "CREDITO", "CRÉDITO":
		return "Crédito"
	case "PENDING", "PENDIENTE":
		return "Pendiente"
	default:
		return pt
	}
}
